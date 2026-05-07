package client

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/google/uuid"
	"github.com/yourusername/go-lanlink/pkg/protocol"
)

// Client represents a lanlinkd client
type Client struct {
	conn          *net.UDPConn
	serverAddr    *net.UDPAddr
	playerID      string
	playerName    string
	roomCode      string
	publicIP      string
	publicPort    int
	messageChan   chan *protocol.Message
	connected     bool
	mu            sync.RWMutex
	done          chan struct{}
}

// Config holds client configuration
type Config struct {
	ServerAddr string // Server address (e.g., "localhost:5555")
	PlayerName string
}

// New creates a new client
func New(config Config) (*Client, error) {
	addr, err := net.ResolveUDPAddr("udp", config.ServerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address: %w", err)
	}

	// Create local connection
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP connection: %w", err)
	}

	return &Client{
		conn:        conn,
		serverAddr:  addr,
		playerID:    uuid.New().String(),
		playerName:  config.PlayerName,
		messageChan: make(chan *protocol.Message, 100),
		done:        make(chan struct{}),
	}, nil
}

// Connect connects to the server
func (c *Client) Connect() error {
	// Start listening for responses
	go c.listen()

	return nil
}

// CreateRoom creates a new room
func (c *Client) CreateRoom() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return "", fmt.Errorf("already connected to a room")
	}

	msg := protocol.NewCreateRoomMessage(c.playerName)
	if err := c.send(msg); err != nil {
		return "", err
	}

	// Wait for response
	response, ok := <-c.messageChan
	if !ok {
		return "", fmt.Errorf("connection closed")
	}

	if response.Type == protocol.TypeRoomCreated {
		c.roomCode = response.RoomCode
		c.publicIP = response.PublicIP
		c.publicPort = response.PublicPort
		c.connected = true
		log.Printf("Room created: %s", c.roomCode)
		if c.publicIP != "" {
			log.Printf("Public address: %s:%d", c.publicIP, c.publicPort)
		}
		return c.roomCode, nil
	}

	if response.Type == protocol.TypeError {
		return "", fmt.Errorf("server error: %s", response.Error)
	}

	return "", fmt.Errorf("unexpected response: %s", response.Type)
}

// JoinRoom joins an existing room
func (c *Client) JoinRoom(roomCode string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("already connected to a room")
	}

	msg := protocol.NewJoinRoomMessage(roomCode, c.playerName)
	if err := c.send(msg); err != nil {
		return err
	}

	// Wait for response
	response, ok := <-c.messageChan
	if !ok {
		return fmt.Errorf("connection closed")
	}

	if response.Type == protocol.TypeRoomJoined {
		c.roomCode = response.RoomCode
		c.playerID = response.PlayerID
		c.connected = true
		log.Printf("Joined room: %s as %s", c.roomCode, c.playerID)
		return nil
	}

	if response.Type == protocol.TypeRoomNotFound {
		return fmt.Errorf("room not found: %s", roomCode)
	}

	if response.Type == protocol.TypeError {
		return fmt.Errorf("server error: %s", response.Error)
	}

	return fmt.Errorf("unexpected response: %s", response.Type)
}

// SendData sends game data to the room
func (c *Client) SendData(payload map[string]interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected to a room")
	}

	msg := protocol.NewGameDataMessage(payload)
	return c.send(msg)
}

// ReceiveMessage returns a channel for incoming messages
func (c *Client) ReceiveMessage() <-chan *protocol.Message {
	return c.messageChan
}

// Close closes the client connection
func (c *Client) Close() {
	close(c.done)
	c.conn.Close()
	close(c.messageChan)
}

// IsConnected returns whether the client is connected to a room
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// RoomCode returns the current room code
func (c *Client) RoomCode() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.roomCode
}

// PlayerID returns the client's player ID
func (c *Client) PlayerID() string {
	return c.playerID
}

// send sends a message to the server
func (c *Client) send(msg *protocol.Message) error {
	data, err := msg.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	_, err = c.conn.WriteToUDP(data, c.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// listen listens for incoming messages from the server
func (c *Client) listen() {
	buffer := make([]byte, 4096)
	for {
		select {
		case <-c.done:
			return
		default:
			n, _, err := c.conn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			msg, err := protocol.Decode(buffer[:n])
			if err != nil {
				log.Printf("Failed to decode message: %v", err)
				continue
			}

			// Route message based on type
			switch msg.Type {
			case protocol.TypeRoomCreated,
			     protocol.TypeRoomJoined,
			     protocol.TypeRoomNotFound,
			     protocol.TypeError:
				// Send to message channel for blocking operations
				select {
				case c.messageChan <- msg:
				default:
					log.Printf("Message channel full, dropping message")
				}

			case protocol.TypeGameData,
			     protocol.TypePlayerJoined,
			     protocol.TypePlayerLeft:
				// These are async notifications
				select {
				case c.messageChan <- msg:
				default:
					log.Printf("Message channel full, dropping message")
				}

			default:
				log.Printf("Unknown message type: %s", msg.Type)
			}
		}
	}
}
