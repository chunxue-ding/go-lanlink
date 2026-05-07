package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/yourusername/go-lanlink/pkg/protocol"
)

// DiscoveredRoom represents a room found on LAN
type DiscoveredRoom struct {
	RoomCode   string
	HostName   string
	Addr       *net.UDPAddr
	Timestamp  int64
}

// DiscoverRooms searches for rooms on LAN
func DiscoverRooms(timeout time.Duration) ([]*DiscoveredRoom, error) {
	// Create UDP connection for listening to broadcasts
	// Use port 0 to let the system assign a random available port
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to listen for discovery: %w", err)
	}
	defer conn.Close()

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(timeout))

	rooms := make(map[string]*DiscoveredRoom)
	buffer := make([]byte, 4096)

	log.Printf("Scanning for rooms on LAN (timeout: %v)...", timeout)

	endTime := time.Now().Add(timeout)
	for time.Now().Before(endTime) {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			// Timeout is expected
			break
		}

		// Parse discovery message
		msg, err := protocol.DecodeDiscoveryMessage(buffer[:n])
		if err != nil {
			log.Printf("Failed to decode discovery message: %v", err)
			continue
		}

		if msg.Type != "lan_broadcast" {
			continue
		}

		// Add room if not already discovered
		if _, exists := rooms[msg.RoomCode]; !exists {
			room := &DiscoveredRoom{
				RoomCode:  msg.RoomCode,
				HostName:  msg.HostName,
				Addr:      addr,
				Timestamp: msg.Timestamp,
			}
			rooms[msg.RoomCode] = room
			log.Printf("Discovered room: %s at %s (host: %s)",
				msg.RoomCode, addr.IP, msg.HostName)
		}
	}

	// Convert map to slice
	result := make([]*DiscoveredRoom, 0, len(rooms))
	for _, room := range rooms {
		result = append(result, room)
	}

	return result, nil
}

// DiscoverAndJoin discovers rooms on LAN and joins the first one
func DiscoverAndJoin(playerName string, timeout time.Duration) (*Client, error) {
	rooms, err := DiscoverRooms(timeout)
	if err != nil {
		return nil, err
	}

	if len(rooms) == 0 {
		return nil, fmt.Errorf("no rooms discovered on LAN")
	}

	// Join the first discovered room
	room := rooms[0]
	log.Printf("Joining discovered room: %s", room.RoomCode)

	// Create client and connect to the discovered room's server
	serverAddr := &net.UDPAddr{
		IP:   room.Addr.IP,
		Port: 5555, // Default server port
	}

	config := Config{
		ServerAddr: serverAddr.String(),
		PlayerName: playerName,
	}

	client, err := New(config)
	if err != nil {
		return nil, err
	}

	if err := client.Connect(); err != nil {
		return nil, err
	}

	if err := client.JoinRoom(room.RoomCode); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

// GetLocalIP returns the local IP address
func GetLocalIP() (net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP, nil
			}
		}
	}

	return nil, fmt.Errorf("no local IP found")
}

// JSON representation of DiscoveredRoom
func (dr *DiscoveredRoom) String() string {
	return fmt.Sprintf("%s (%s) at %s", dr.RoomCode, dr.HostName, dr.Addr.IP)
}

// ToJSON converts DiscoveredRoom to JSON
func (dr *DiscoveredRoom) ToJSON() (string, error) {
	data, err := json.Marshal(dr)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
