package artnet

import (
	"fmt"
	"net"
	"strings"
)

const (
	// addressRange specifies the network CIDR an art-net network should have.
	addressRange = "192.168.6.0/24"
)

// FindArtNetIP finds the matching interface with an IP address inside the addressRange.
func FindArtNetIP() (net.IP, error) {
	_, cidrNet, _ := net.ParseCIDR(addressRange)
	address, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("error getting ips: %w", err)
	}

	for _, addr := range address {
		ip := addr.(*net.IPNet).IP

		if strings.Contains(ip.String(), ":") {
			continue
		}

		if cidrNet.Contains(ip) {
			return ip, nil
		}
	}

	return nil, nil
}
