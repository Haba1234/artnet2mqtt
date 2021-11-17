package artnet

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"artnet2mqtt/internal/clientmqtt"
	"artnet2mqtt/internal/logger"
	"github.com/Haba1234/go-artnet"
)

// ArtNet is transport for the ArtNet protocol (DMX over UDP/IP).
type ArtNet struct {
	logger      logger.Logger
	client      *clientmqtt.ClientMQTT
	sender      *artnet.Controller
	state       *State
	sendTrigger chan UniverseStateMap
	ctx         context.Context
	dmxDataCh   <-chan clientmqtt.DataCh
}

// Controller is a convenience interface to use within this application.
type Controller interface {
	SetDMXChannelValue(value ChannelValue)
	SetDMXChannelValues(values []ChannelValue)
	Start(ctx context.Context, dmxDataCh <-chan clientmqtt.DataCh)
	Stop()
}

// NewController returns an art-net Controller as an anonymous interface.
func NewController(log logger.Logger, client *clientmqtt.ClientMQTT) (*ArtNet, error) {
	ip, err := FindArtNetIP()
	if err != nil {
		return nil, fmt.Errorf("failed to find the art-net IP: %w", err)
	}

	if len(ip) == 0 {
		return nil, errors.New("failed to find the art-net IP: No interface found")
	}

	host, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve hostname: %w", err)
	}

	host = strings.ToLower(strings.Split(host, ".")[0])
	log.With(logger.Fields{"module": "Art-net"}).Infof("Using ArtNet IP %s and hostname %s", ip.String(), host)

	senderLogger := artnet.NewDefaultLogger("info")

	control := &ArtNet{
		logger:      log,
		client:      client,
		sender:      artnet.NewController(host, ip, senderLogger, artnet.MaxFPS(1)),
		state:       NewState(),
		sendTrigger: make(chan UniverseStateMap, 100),
	}

	return control, nil
}

// Start the ArtNet.
func (c *ArtNet) Start(ctx context.Context, dmxDataCh <-chan clientmqtt.DataCh) error {
	if err := c.sender.Start(); err != nil {
		return fmt.Errorf("failed to start Controller: %w", err)
	}

	c.ctx = ctx
	c.dmxDataCh = dmxDataCh
	go c.sendBackground()
	go c.debugDevices()
	go c.dataProcessing()
	return nil
}

// Stop the ArtNet.
func (c *ArtNet) Stop() {
	close(c.sendTrigger)
	c.sender.Stop()
}

func (c *ArtNet) SetDMXChannelValue(value ChannelValue) {
	c.state.SetChannel(value.Universe, value.Channel, value.Value)
	c.triggerSend()
}

func (c *ArtNet) SetDMXChannelValues(values []ChannelValue) {
	c.state.SetChannelValues(values)
	c.triggerSend()
}

func (c *ArtNet) triggerSend() {
	c.logger.With(logger.Fields{"module": "art-net"}).Debug("DMX. Отправка в канал")
	c.sendTrigger <- c.state.Get()
}

func (c *ArtNet) sendBackground() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case data := <-c.sendTrigger:
			for u, dmx := range data {
				// u - адрес.
				// dmx - массив данных до 512 байт.
				c.logger.With(logger.Fields{"module": "art-net"}).Debugf("DMX. Отправка в контроллер по адресу %v\n", u)
				c.sender.SendDMXToAddress(dmx.toByteSlice(), c.universeToAddress(u))
			}
		}
	}
}

func (c *ArtNet) dataProcessing() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case d := <-c.dmxDataCh:
			go func() {
				d := d
				dmxData := make([]ChannelValue, len(d.Data))
				for i, v := range d.Data {
					dmxData[i] = ChannelValue{d.Addr, v.Channel, v.Value}
				}
				c.logger.With(logger.Fields{"module": "art-net"}).Debug("DMX. Данные пришли с MQTT")
				c.SetDMXChannelValues(dmxData)
			}()
		}
	}
}

// universeToAddress converts a dmx universe to art-net address
// universe: старший байт - SubUni, младший байт - Net.
func (c *ArtNet) universeToAddress(universe uint16) artnet.Address {
	v := make([]uint8, 2)
	binary.BigEndian.PutUint16(v, universe)

	return artnet.Address{
		Net:    v[0],
		SubUni: v[1],
	}
}

// NodeToString returns a string representation of the given Node.
func NodeToString(n *artnet.ControlledNode) (string, NodeTopic) {
	var inputs, outputs []string
	var out []uint16
	var outStr []string
	// TODO переделать массивы.
	for _, p := range n.Node.InputPorts {
		inputs = append(inputs, fmt.Sprintf("%s: %s", p.Address.String(), p.Type.String()))
	}

	for _, p := range n.Node.OutputPorts {
		outputs = append(outputs, fmt.Sprintf("%s: %s", p.Address.String(), p.Type.String()))
		out = append(out, uint16(p.Address.Integer()))
		outStr = append(outStr, p.Address.String())
	}

	return fmt.Sprintf(
			" | IP=%s name=%q type=%q manufacturer=%q desc=%q inputs=%q outputs=%q",
			n.UDPAddress.String(), n.Node.Name, n.Node.Type,
			n.Node.Manufacturer, n.Node.Description,
			strings.Join(inputs, "; "), strings.Join(outputs, "; "),
		), NodeTopic{
			Name:      n.Node.Name,
			OutputStr: outStr,
			Output:    out,
		}
}

func ips(nodes []*artnet.ControlledNode) (ips IpsType) {
	ips = IpsType{}
	for _, n := range nodes {
		node, out := NodeToString(n)
		ips.Ips = append(ips.Ips, node)
		ips.Topics = append(ips.Topics, out)
	}
	return ips
}

func (c *ArtNet) debugDevices() {
	t := time.NewTicker(30 * time.Second)
	for range t.C {
		l := len(c.sender.Nodes) // Кол-во видимых узлов.
		dev := ips(c.sender.Nodes)
		c.logger.With(logger.Fields{"module": "art-net"}).Debugf("Currently %d devices are registered: %v\n", l, dev.Ips)
		for _, top := range dev.Topics {
			for _, out := range top.Output {
				nameTopic := fmt.Sprintf("artnet/%s.%v", top.Name, out)
				c.logger.With(logger.Fields{"module": "art-net"}).Debug("Publication. nameTopic:", nameTopic)
				c.client.PubTopic(nameTopic, out)
			}
		}
	}
}
