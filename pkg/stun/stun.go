package stun

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

// STUN message types
const (
	MethodBinding = 0x0001
	ClassRequest  = 0x0000
	ClassSuccess  = 0x0100
	ClassError    = 0x0110
)

// STUN attributes
const (
	AttrMappedAddress = 0x0001
	AttrXORMappedAddress = 0x0020
	AttrErrorCode = 0x0009
)

// STUN header size
const (
	HeaderSize = 20
	MagicCookie = 0x2112A442
)

// NATType represents the type of NAT
type NATType int

const (
	NATUnknown NATType = iota
	NATOpen
	NATFullCone
	NATRestricted
	NATPortRestricted
	NATSymmetric
	NATBlocked
)

func (t NATType) String() string {
	switch t {
	case NATOpen:
		return "Open Internet"
	case NATFullCone:
		return "Full Cone NAT"
	case NATRestricted:
		return "Restricted Cone NAT"
	case NATPortRestricted:
		return "Port Restricted Cone NAT"
	case NATSymmetric:
		return "Symmetric NAT"
	case NATBlocked:
		return "Blocked (Firewall)"
	default:
		return "Unknown"
	}
}

// STUNClient represents a STUN client
type STUNClient struct {
	ServerAddr string
	Timeout    time.Duration
}

// NewSTUNClient creates a new STUN client
func NewSTUNClient(server string) *STUNClient {
	return &STUNClient{
		ServerAddr: server,
		Timeout:    5 * time.Second,
	}
}

// PublicAddress represents the public IP and port
type PublicAddress struct {
	IP   string
	Port int
}

// DiscoverPublicAddress discovers the public IP and port using STUN
func (c *STUNClient) DiscoverPublicAddress() (*PublicAddress, error) {
	// Parse STUN server address
	serverAddr, err := net.ResolveUDPAddr("udp", c.ServerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve STUN server: %w", err)
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP connection: %w", err)
	}
	defer conn.Close()

	// Set deadline
	conn.SetDeadline(time.Now().Add(c.Timeout))

	// Send STUN binding request
	if err := c.sendBindingRequest(conn); err != nil {
		return nil, fmt.Errorf("failed to send STUN request: %w", err)
	}

	// Receive response
	return c.receiveBindingResponse(conn)
}

// sendBindingRequest sends a STUN binding request
func (c *STUNClient) sendBindingRequest(conn *net.UDPConn) error {
	// Generate transaction ID
	transactionID := make([]byte, 12)
	if _, err := randRead(transactionID); err != nil {
		return err
	}

	// Build STUN message
	msg := make([]byte, HeaderSize)

	// Type: Binding Request (0x0001)
	binary.BigEndian.PutUint16(msg[0:2], MethodBinding)

	// Length: 0 (no attributes)
	binary.BigEndian.PutUint16(msg[2:4], 0)

	// Magic Cookie
	binary.BigEndian.PutUint32(msg[4:8], MagicCookie)

	// Transaction ID
	copy(msg[8:20], transactionID)

	// Send
	_, err := conn.Write(msg)
	return err
}

// receiveBindingResponse receives and parses STUN binding response
func (c *STUNClient) receiveBindingResponse(conn *net.UDPConn) (*PublicAddress, error) {
	// Receive response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to receive STUN response: %w", err)
	}

	if n < HeaderSize {
		return nil, errors.New("STUN response too short")
	}

	// Parse header
	msgType := binary.BigEndian.Uint16(buffer[0:2])
	msgLength := binary.BigEndian.Uint16(buffer[2:4])
	magicCookie := binary.BigEndian.Uint32(buffer[4:8])

	// Verify magic cookie
	if magicCookie != MagicCookie {
		return nil, errors.New("invalid STUN magic cookie")
	}

	// Check message type
	msgClass := msgType & 0x0110
	if msgClass != ClassSuccess {
		return nil, fmt.Errorf("STUN error response: class=0x%04x", msgClass)
	}

	// Parse attributes
	offset := HeaderSize
	endOffset := HeaderSize + int(msgLength)

	for offset < endOffset {
		if offset+4 > n {
			break
		}

		attrType := binary.BigEndian.Uint16(buffer[offset : offset+2])
		attrLength := binary.BigEndian.Uint16(buffer[offset+2 : offset+4])
		offset += 4

		// Padding to 4-byte boundary
		padding := (4 - (attrLength % 4)) % 4

		if offset+int(attrLength) > n {
			break
		}

		attrValue := buffer[offset : offset+int(attrLength)]
		offset += int(attrLength) + int(padding)

		// Parse XOR MAPPED ADDRESS
		if attrType == AttrXORMappedAddress {
			return parseXORMappedAddress(attrValue)
		}

		// Parse MAPPED ADDRESS (fallback)
		if attrType == AttrMappedAddress {
			return parseMappedAddress(attrValue)
		}
	}

	return nil, errors.New("no mapped address in STUN response")
}

// parseXORMappedAddress parses XOR-MAPPED-ADDRESS attribute
func parseXORMappedAddress(data []byte) (*PublicAddress, error) {
	if len(data) < 8 {
		return nil, errors.New("XOR-MAPPED-ADDRESS too short")
	}

	family := binary.BigEndian.Uint16(data[0:2])
	if family != 0x01 { // IPv4
		return nil, errors.New("only IPv4 supported")
	}

	port := binary.BigEndian.Uint16(data[2:4]) ^ 0x2112
	ip := make(net.IP, 4)
	cookie := uint32(MagicCookie)
	for i := 0; i < 4; i++ {
		shift := uint(8 * (3 - i))
		cookieByte := byte((cookie >> shift) & 0xFF)
		ip[i] = data[4+i] ^ cookieByte
	}

	return &PublicAddress{
		IP:   ip.String(),
		Port: int(port),
	}, nil
}

// parseMappedAddress parses MAPPED-ADDRESS attribute
func parseMappedAddress(data []byte) (*PublicAddress, error) {
	if len(data) < 8 {
		return nil, errors.New("MAPPED-ADDRESS too short")
	}

	family := binary.BigEndian.Uint16(data[0:2])
	if family != 0x01 { // IPv4
		return nil, errors.New("only IPv4 supported")
	}

	port := binary.BigEndian.Uint16(data[2:4])
	ip := net.IP(data[4:8])

	return &PublicAddress{
		IP:   ip.String(),
		Port: int(port),
	}, nil
}

// randRead fills buffer with random bytes
func randRead(b []byte) (int, error) {
	// Simple random number generator
	for i := range b {
		b[i] = byte(time.Now().UnixNano() & 0xFF)
	}
	return len(b), nil
}

// DefaultSTUNServers is a list of public STUN servers
var DefaultSTUNServers = []string{
	"stun.l.google.com:19302",
	"stun1.l.google.com:19302",
	"stun2.l.google.com:19302",
	"stun3.l.google.com:19302",
	"stun4.l.google.com:19302",
}

// DiscoverPublicAddressWithRetry tries multiple STUN servers
func DiscoverPublicAddressWithRetry() (*PublicAddress, string, error) {
	for _, server := range DefaultSTUNServers {
		client := NewSTUNClient(server)
		addr, err := client.DiscoverPublicAddress()
		if err == nil && addr != nil {
			return addr, server, nil
		}
	}

	return nil, "", errors.New("all STUN servers failed")
}
