package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"artnet2mqtt/internal/artnet"
	"artnet2mqtt/internal/clientmqtt"
	"artnet2mqtt/internal/config"
	"artnet2mqtt/internal/logger"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "configs/conf.toml", "Path to configuration file")
}

func main() {
	flag.Parse()
	cfg, err := config.NewConfig(configFile)
	if err != nil {
		fmt.Printf("configuration file read error: %v", err)
		os.Exit(1)
	}

	log, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		fmt.Printf("failed to create a logger: %v", err)
		os.Exit(1)
	}

	log.With(logger.Fields{"module": "logger"}).Debug("newLogger created ok")

	client := clientmqtt.NewClient(log, ConvertConfigClientMQTT(cfg.MQTT))
	log.With(logger.Fields{"module": "mqtt"}).Debug("NewClient created ok")

	a, err := artnet.NewController(log, client)
	if err != nil {
		log.With(logger.Fields{"module": "art-net"}).Errorf("error while creating a new controller art-net. %v", err)
		os.Exit(1)
	}
	log.With(logger.Fields{"module": "art-net"}).Debug("NewController created ok")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	// Канал для передачи.
	dmxDataCh := make(chan clientmqtt.DataCh, 10)

	if err = a.Start(ctx, dmxDataCh); err != nil {
		log.Error("failed to start art-net service:", err.Error())
		cancel()
	}

	if err = client.Start(ctx, dmxDataCh); err != nil {
		log.Error("failed to start MQTT service:", err.Error())
		cancel()
	}

	<-ctx.Done()

	if err := client.Stop(); err != nil {
		log.Error("failed to stop MQTT service:", err.Error())
	}

	a.Stop()

	close(dmxDataCh)

	log.Info("shutdown complete")
}

// ConvertConfigClientMQTT преобразует структуры.
func ConvertConfigClientMQTT(cfg config.MQTTConf) clientmqtt.MQTTConf {
	return clientmqtt.MQTTConf{
		ClientID: cfg.ClientID,
		Schema:   "tcp",
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
	}
}
