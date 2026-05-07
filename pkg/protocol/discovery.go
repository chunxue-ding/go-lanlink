package protocol

import (
	"encoding/json"
	"net"
	"time"
)

const (
	// DiscoveryPort is the UDP port for LAN discovery
	DiscoveryPort = 5556

	// BroadcastInterval is how often to broadcast room info
	BroadcastInterval = 1 * time.Second

	// DiscoveryTimeout is how long to wait for room discovery
	DiscoveryTimeout = 3 * time.Second
)

// DiscoveryMessage represents a LAN discovery broadcast
type DiscoveryMessage struct {
	Type      string `json:"type"`
	RoomCode  string `json:"room_code"`
	HostName  string `json:"host_name,omitempty"`
	Version   string `json:"version,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// Encode converts DiscoveryMessage to JSON
func (dm *DiscoveryMessage) Encode() ([]byte, error) {
	return json.Marshal(dm)
}

// DecodeDiscoveryMessage parses JSON into DiscoveryMessage
func DecodeDiscoveryMessage(data []byte) (*DiscoveryMessage, error) {
	var dm DiscoveryMessage
	err := json.Unmarshal(data, &dm)
	if err != nil {
		return nil, err
	}
	return &dm, nil
}

// NewDiscoveryMessage creates a new discovery broadcast message
func NewDiscoveryMessage(roomCode, hostName string) *DiscoveryMessage {
	return &DiscoveryMessage{
		Type:      "lan_broadcast",
		RoomCode:  roomCode,
		HostName:  hostName,
		Version:   "1.0.0",
		Timestamp: time.Now().Unix(),
	}
}

// GetBroadcastAddresses returns all broadcast addresses for LAN discovery
func GetBroadcastAddresses() ([]*net.UDPAddr, error) {
	addresses := []*net.UDPAddr{}

	// Add standard broadcast address (works for most networks)
	broadcastAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:5556")
	if err != nil {
		return nil, err
	}
	addresses = append(addresses, broadcastAddr)

	// Try to add local network-specific broadcast addresses
	interfaces, err := net.Interfaces()
	if err != nil {
		return addresses, nil // Return at least the standard broadcast
	}

	for _, iface := range interfaces {
		// Skip down interfaces and loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ipNet *net.IPNet
			switch v := addr.(type) {
			case *net.IPNet:
				ipNet = v
			case *net.IPAddr:
				ipNet = &net.IPNet{IP: v.IP, Mask: v.IP.DefaultMask()}
			}

			if ipNet == nil || ipNet.IP.To4() == nil {
				continue // Skip IPv6
			}

			// Calculate broadcast address
			broadcastIP := make(net.IP, 4)
			for i := range ipNet.IP.To4() {
				broadcastIP[i] = ipNet.IP.To4()[i] | ^ipNet.Mask[i]
			}

			broadcastAddr, err := net.ResolveUDPAddr("udp",
				broadcastIP.String()+":5556")
			if err == nil {
				// Avoid duplicates
				isDuplicate := false
				for _, existing := range addresses {
					if existing.IP.Equal(broadcastIP) {
						isDuplicate = true
						break
					}
				}
				if !isDuplicate {
					addresses = append(addresses, broadcastAddr)
				}
			}
		}
	}

	return addresses, nil
}
