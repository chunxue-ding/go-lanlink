package relay

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/go-lanlink/pkg/protocol"
)

// Session represents a relayed game session
type Session struct {
	ID        string
	RoomCode  string
	Clients   map[string]*RelayClient
	CreatedAt time.Time
	mu        sync.RWMutex
}

// RelayClient represents a connected client
type RelayClient struct {
	ID       string
	Name     string
	Addr     *net.UDPAddr
	Conn     *net.UDPConn
	RoomCode string
	LastSeen time.Time
}

// RelayServer represents the relay server
type RelayServer struct {
	conn       *net.UDPConn
	port       int
	sessions   map[string]*Session // room_code -> session
	clients    map[string]*RelayClient // client_id -> client
	clientsByAddr map[string]*RelayClient // addr.String() -> client
	mu         sync.RWMutex
	stats      RelayStats
	done       chan struct{}
}

// RelayStats holds relay statistics
type RelayStats struct {
	TotalSessions    int64
	TotalClients     int64
	BytesForwarded   int64
	PacketsForwarded int64
}

// Config holds relay server configuration
type Config struct {
	Port           int
	SessionTimeout time.Duration
	ClientTimeout  time.Duration
}

// DefaultConfig returns default relay configuration
func DefaultConfig() Config {
	return Config{
		Port:           7000,
		SessionTimeout: 1 * time.Hour,
		ClientTimeout:  5 * time.Minute,
	}
}

// New creates a new relay server
func New(config Config) (*RelayServer, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	return &RelayServer{
		conn:         conn,
		port:         config.Port,
		sessions:     make(map[string]*Session),
		clients:      make(map[string]*RelayClient),
		clientsByAddr: make(map[string]*RelayClient),
		done:         make(chan struct{}),
	}, nil
}

// Start starts the relay server
func (r *RelayServer) Start() {
	log.Printf("Relay server started on port %d", r.port)

	// Start cleanup goroutine
	go r.cleanup()

	// Handle incoming packets
	buffer := make([]byte, 4096)
	for {
		select {
		case <-r.done:
			return
		default:
			n, addr, err := r.conn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			go r.handlePacket(buffer[:n], addr)
		}
	}
}

// Stop stops the relay server
func (r *RelayServer) Stop() {
	close(r.done)
	r.conn.Close()
}

// handlePacket processes incoming packets
func (r *RelayServer) handlePacket(data []byte, addr *net.UDPAddr) {
	// Try to parse as protocol message first
	msg, err := protocol.Decode(data)
	if err == nil {
		r.handleMessage(msg, addr)
		return
	}

	// If not a protocol message, treat as raw game data
	r.forwardGameData(data, addr)
}

// handleMessage handles protocol messages
func (r *RelayServer) handleMessage(msg *protocol.Message, addr *net.UDPAddr) {
	switch msg.Type {
	case protocol.TypeCreateRoom:
		r.handleCreateRoom(msg, addr)
	case protocol.TypeJoinRoom:
		r.handleJoinRoom(msg, addr)
	case protocol.TypeGameData:
		// Forward protocol messages
		msgData, _ := msg.Encode()
		r.forwardGameData(msgData, addr)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// handleCreateRoom creates a new relay session
func (r *RelayServer) handleCreateRoom(msg *protocol.Message, addr *net.UDPAddr) {
	// Generate room code
	roomCode, err := protocol.GenerateRoomCode()
	if err != nil {
		r.sendError(addr, "Failed to generate room code")
		return
	}

	// Create session
	session := &Session{
		ID:        uuid.New().String(),
		RoomCode:  roomCode,
		Clients:   make(map[string]*RelayClient),
		CreatedAt: time.Now(),
	}

	// Create host client
	hostClient := &RelayClient{
		ID:       uuid.New().String(),
		Name:     msg.PlayerName,
		Addr:     addr,
		Conn:     r.conn,
		RoomCode: roomCode,
		LastSeen: time.Now(),
	}

	session.Clients[hostClient.ID] = hostClient

	// Store
	r.mu.Lock()
	r.sessions[roomCode] = session
	r.clients[hostClient.ID] = hostClient
	r.clientsByAddr[addr.String()] = hostClient
	r.stats.TotalSessions++
	r.stats.TotalClients++
	r.mu.Unlock()

	// Send response
	response := protocol.NewRoomCreatedMessage(roomCode)
	response.PublicAddress = fmt.Sprintf("%s:%d", getPublicIP(), r.port)
	r.sendMessage(response, addr)

	log.Printf("[RELAY] Room created: %s by %s (%s)", roomCode, msg.PlayerName, hostClient.ID)
}

// handleJoinRoom joins an existing relay session
func (r *RelayServer) handleJoinRoom(msg *protocol.Message, addr *net.UDPAddr) {
	roomCode := protocol.NormalizeRoomCode(msg.RoomCode)

	// Find session
	r.mu.RLock()
	session, exists := r.sessions[roomCode]
	r.mu.RUnlock()

	if !exists {
		response := protocol.NewRoomNotFoundMessage(roomCode)
		r.sendMessage(response, addr)
		log.Printf("[RELAY] Room not found: %s", roomCode)
		return
	}

	// Create client
	client := &RelayClient{
		ID:       uuid.New().String(),
		Name:     msg.PlayerName,
		Addr:     addr,
		Conn:     r.conn,
		RoomCode: roomCode,
		LastSeen: time.Now(),
	}

	// Add to session
	session.mu.Lock()
	session.Clients[client.ID] = client
	session.mu.Unlock()

	// Store
	r.mu.Lock()
	r.clients[client.ID] = client
	r.clientsByAddr[addr.String()] = client
	r.stats.TotalClients++
	r.mu.Unlock()

	// Send response
	response := protocol.NewRoomJoinedMessage(roomCode, client.ID)
	response.PublicAddress = fmt.Sprintf("%s:%d", getPublicIP(), r.port)
	r.sendMessage(response, addr)

	// Notify others
	notifyMsg := protocol.NewPlayerJoinedMessage(client.ID, client.Name)
	r.broadcastToSession(session, notifyMsg, addr)

	log.Printf("[RELAY] Player %s (%s) joined room %s", msg.PlayerName, client.ID, roomCode)
}

// forwardGameData forwards game data to other clients in the session
func (r *RelayServer) forwardGameData(data []byte, senderAddr *net.UDPAddr) {
	// Find client
	r.mu.RLock()
	client, exists := r.clientsByAddr[senderAddr.String()]
	r.mu.RUnlock()

	if !exists {
		return
	}

	// Find session
	r.mu.RLock()
	session, exists := r.sessions[client.RoomCode]
	r.mu.RUnlock()

	if !exists {
		return
	}

	// Update last seen
	client.LastSeen = time.Now()

	// Forward to all other clients
	session.mu.RLock()
	defer session.mu.RUnlock()

	for _, otherClient := range session.Clients {
		if otherClient.ID != client.ID {
			r.conn.WriteToUDP(data, otherClient.Addr)
			r.stats.BytesForwarded += int64(len(data))
			r.stats.PacketsForwarded++
		}
	}
}

// broadcastToSession sends a message to all clients in a session
func (r *RelayServer) broadcastToSession(session *Session, msg *protocol.Message, excludeAddr *net.UDPAddr) {
	data, err := msg.Encode()
	if err != nil {
		return
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	for _, client := range session.Clients {
		if excludeAddr == nil || client.Addr.String() != excludeAddr.String() {
			r.conn.WriteToUDP(data, client.Addr)
		}
	}
}

// sendMessage sends a message to a specific address
func (r *RelayServer) sendMessage(msg *protocol.Message, addr *net.UDPAddr) {
	data, err := msg.Encode()
	if err != nil {
		return
	}

	r.conn.WriteToUDP(data, addr)
}

// sendError sends an error message
func (r *RelayServer) sendError(addr *net.UDPAddr, errMsg string) {
	msg := protocol.NewErrorMessage(errMsg)
	r.sendMessage(msg, addr)
}

// cleanup removes stale sessions and clients
func (r *RelayServer) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			r.cleanupStale()
		}
	}
}

// cleanupStale removes stale sessions and clients
func (r *RelayServer) cleanupStale() {
	now := time.Now()

	// Clean up stale sessions
	r.mu.Lock()
	defer r.mu.Unlock()

	for roomCode, session := range r.sessions {
		session.mu.RLock()
		allStale := true
		for _, client := range session.Clients {
			if now.Sub(client.LastSeen) < 5*time.Minute {
				allStale = false
				break
			}
		}
		session.mu.RUnlock()

		if allStale || now.Sub(session.CreatedAt) > 1*time.Hour {
			// Remove session
			for _, client := range session.Clients {
				delete(r.clients, client.ID)
				delete(r.clientsByAddr, client.Addr.String())
			}
			delete(r.sessions, roomCode)
			log.Printf("[RELAY] Cleaned up stale session: %s", roomCode)
		}
	}
}

// GetStats returns current relay statistics
func (r *RelayServer) GetStats() RelayStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy
	return RelayStats{
		TotalSessions:    r.stats.TotalSessions,
		TotalClients:     r.stats.TotalClients,
		BytesForwarded:   r.stats.BytesForwarded,
		PacketsForwarded: r.stats.PacketsForwarded,
	}
}

// getPublicIP returns the public IP address
func getPublicIP() string {
	// Try to get local IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "localhost"
}
