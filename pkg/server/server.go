package server

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/go-lanlink/pkg/protocol"
	"github.com/yourusername/go-lanlink/pkg/stun"
)

// Player represents a connected player
type Player struct {
	ID       string
	Name     string
	Addr     *net.UDPAddr
	Conn     *net.UDPConn
	IsHost   bool
	RoomCode string
}

// Room represents a game room
type Room struct {
	Code         string
	Host         *Player
	Players      map[string]*Player
	GameData     chan *protocol.Message
	PublicIP     string
	PublicPort   int
	mu           sync.RWMutex
}

// Server represents the lanlinkd server
type Server struct {
	conn              *net.UDPConn
	broadcastConn     *net.UDPConn
	port              int
	rooms             map[string]*Room
	players           map[string]*Player
	playersByAddr     map[string]*Player
	mu                sync.RWMutex
	done              chan struct{}
	enableLanDiscovery bool
}

// New creates a new server
func New(port int) (*Server, error) {
	return NewWithDiscovery(port, true)
}

// NewWithDiscovery creates a new server with optional LAN discovery
func NewWithDiscovery(port int, enableDiscovery bool) (*Server, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP: %w", err)
	}

	// Get actual port (in case port was 0)
	actualPort := conn.LocalAddr().(*net.UDPAddr).Port

	server := &Server{
		conn:               conn,
		port:               actualPort,
		rooms:              make(map[string]*Room),
		players:            make(map[string]*Player),
		playersByAddr:      make(map[string]*Player),
		done:               make(chan struct{}),
		enableLanDiscovery: enableDiscovery,
	}

	// Setup broadcast connection for LAN discovery
	if enableDiscovery {
		broadcastAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", protocol.DiscoveryPort))
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to resolve broadcast address: %w", err)
		}

		server.broadcastConn, err = net.ListenUDP("udp", broadcastAddr)
		if err != nil {
			log.Printf("Warning: Failed to setup LAN discovery: %v", err)
			server.enableLanDiscovery = false
		}
	}

	return server, nil
}

// Start starts the server
func (s *Server) Start() {
	log.Printf("Server started on port %d", s.port)
	if s.enableLanDiscovery {
		log.Printf("LAN discovery enabled on port %d", protocol.DiscoveryPort)
	}

	// Start room data forwarding
	go s.forwardRoomData()

	// Start LAN discovery broadcast if enabled
	if s.enableLanDiscovery {
		go s.broadcastLanDiscovery()
	}

	// Handle incoming messages
	buffer := make([]byte, 4096)
	for {
		select {
		case <-s.done:
			return
		default:
			n, addr, err := s.conn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			go s.handleMessage(buffer[:n], addr)
		}
	}
}

// Stop stops the server
func (s *Server) Stop() {
	close(s.done)
	if s.broadcastConn != nil {
		s.broadcastConn.Close()
	}
	s.conn.Close()
}

// Port returns the server's listening port
func (s *Server) Port() int {
	return s.port
}

// handleMessage processes incoming messages
func (s *Server) handleMessage(data []byte, addr *net.UDPAddr) {
	msg, err := protocol.Decode(data)
	if err != nil {
		log.Printf("Failed to decode message: %v", err)
		return
	}

	log.Printf("Received message type: %s from %s", msg.Type, addr)

	switch msg.Type {
	case protocol.TypeCreateRoom:
		s.handleCreateRoom(msg, addr)
	case protocol.TypeJoinRoom:
		s.handleJoinRoom(msg, addr)
	case protocol.TypeGameData:
		s.handleGameData(msg, addr)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// handleCreateRoom creates a new room
func (s *Server) handleCreateRoom(msg *protocol.Message, addr *net.UDPAddr) {
	// Generate room code
	roomCode, err := protocol.GenerateRoomCode()
	if err != nil {
		s.sendError(addr, "Failed to generate room code")
		return
	}

	// Create player
	playerID := uuid.New().String()
	player := &Player{
		ID:       playerID,
		Name:     msg.PlayerName,
		Addr:     addr,
		Conn:     s.conn,
		IsHost:   true,
		RoomCode: roomCode,
	}

	// Discover public IP using STUN
	var publicIP string
	var publicPort int
	log.Println("Discovering public IP address using STUN...")
	publicAddr, serverUsed, err := stun.DiscoverPublicAddressWithRetry()
	if err != nil {
		log.Printf("STUN discovery failed: %v (using local address)", err)
		// Fall back to local IP
		publicIP = addr.IP.String()
		publicPort = s.port
	} else {
		publicIP = publicAddr.IP
		publicPort = publicAddr.Port
		log.Printf("Public address discovered: %s:%d (via %s)", publicIP, publicPort, serverUsed)
	}

	// Create room
	room := &Room{
		Code:       roomCode,
		Host:       player,
		Players:    make(map[string]*Player),
		GameData:   make(chan *protocol.Message, 100),
		PublicIP:   publicIP,
		PublicPort: publicPort,
	}
	room.Players[playerID] = player

	// Store room and player
	s.mu.Lock()
	s.rooms[roomCode] = room
	s.players[playerID] = player
	s.playersByAddr[addr.String()] = player
	s.mu.Unlock()

	// Send response with public address
	response := protocol.NewRoomCreatedMessage(roomCode)
	response.PublicIP = publicIP
	response.PublicPort = publicPort
	response.PublicAddress = fmt.Sprintf("%s:%d", publicIP, publicPort)
	s.sendMessage(response, addr)

	log.Printf("Room created: %s by %s (%s)", roomCode, player.Name, player.ID)
	log.Printf("Room public address: %s:%d", publicIP, publicPort)
}

// handleJoinRoom joins an existing room
func (s *Server) handleJoinRoom(msg *protocol.Message, addr *net.UDPAddr) {
	roomCode := protocol.NormalizeRoomCode(msg.RoomCode)

	// Find room
	s.mu.RLock()
	room, exists := s.rooms[roomCode]
	s.mu.RUnlock()

	if !exists {
		response := protocol.NewRoomNotFoundMessage(roomCode)
		s.sendMessage(response, addr)
		log.Printf("Room not found: %s", roomCode)
		return
	}

	// Create player
	playerID := uuid.New().String()
	player := &Player{
		ID:       playerID,
		Name:     msg.PlayerName,
		Addr:     addr,
		Conn:     s.conn,
		IsHost:   false,
		RoomCode: roomCode,
	}

	// Add to room
	room.mu.Lock()
	room.Players[playerID] = player
	room.mu.Unlock()

	// Store player
	s.mu.Lock()
	s.players[playerID] = player
	s.playersByAddr[addr.String()] = player
	s.mu.Unlock()

	// Send response to joining player
	response := protocol.NewRoomJoinedMessage(roomCode, playerID)
	response.PublicIP = room.PublicIP
	response.PublicPort = room.PublicPort
	response.PublicAddress = fmt.Sprintf("%s:%d", room.PublicIP, room.PublicPort)
	s.sendMessage(response, addr)

	// Notify all players in room
	notifyMsg := protocol.NewPlayerJoinedMessage(playerID, player.Name)
	s.broadcastToRoom(room, notifyMsg, addr)

	log.Printf("Player %s (%s) joined room %s (host: %s:%d)",
		player.Name, player.ID, roomCode, room.PublicIP, room.PublicPort)
}

// handleGameData processes game data and forwards to room members
func (s *Server) handleGameData(msg *protocol.Message, addr *net.UDPAddr) {
	// Find player
	s.mu.RLock()
	player, exists := s.playersByAddr[addr.String()]
	s.mu.RUnlock()

	if !exists {
		log.Printf("Unknown player address: %s", addr)
		return
	}

	// Find room
	s.mu.RLock()
	room, exists := s.rooms[player.RoomCode]
	s.mu.RUnlock()

	if !exists {
		log.Printf("Room not found for player: %s", player.ID)
		return
	}

	// Forward to room members (excluding sender)
	room.mu.RLock()
	players := make([]*Player, 0, len(room.Players))
	for _, p := range room.Players {
		if p.ID != player.ID {
			players = append(players, p)
		}
	}
	room.mu.RUnlock()

	// Send to each player
	for _, p := range players {
		s.sendMessage(msg, p.Addr)
	}
}

// forwardRoomData forwards game data between room members
func (s *Server) forwardRoomData() {
	// This is a placeholder for future optimization
	// Currently we forward immediately in handleGameData
}

// broadcastToRoom sends a message to all players in a room
func (s *Server) broadcastToRoom(room *Room, msg *protocol.Message, excludeAddr *net.UDPAddr) {
	room.mu.RLock()
	defer room.mu.RUnlock()

	for _, player := range room.Players {
		if excludeAddr == nil || player.Addr.String() != excludeAddr.String() {
			s.sendMessage(msg, player.Addr)
		}
	}
}

// sendMessage sends a message to a specific address
func (s *Server) sendMessage(msg *protocol.Message, addr *net.UDPAddr) {
	data, err := msg.Encode()
	if err != nil {
		log.Printf("Failed to encode message: %v", err)
		return
	}

	_, err = s.conn.WriteToUDP(data, addr)
	if err != nil {
		log.Printf("Failed to send message to %s: %v", addr, err)
	}
}

// sendError sends an error message
func (s *Server) sendError(addr *net.UDPAddr, errMsg string) {
	msg := protocol.NewErrorMessage(errMsg)
	s.sendMessage(msg, addr)
}

// broadcastLanDiscovery broadcasts room info to LAN
func (s *Server) broadcastLanDiscovery() {
	ticker := time.NewTicker(protocol.BroadcastInterval)
	defer ticker.Stop()

	// Get broadcast addresses
	addresses, err := protocol.GetBroadcastAddresses()
	if err != nil {
		log.Printf("Failed to get broadcast addresses: %v", err)
		return
	}

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			// Broadcast all rooms
			s.mu.RLock()
			for roomCode, room := range s.rooms {
				// Create discovery message
				discoveryMsg := protocol.NewDiscoveryMessage(roomCode, room.Host.Name)
				data, err := discoveryMsg.Encode()
				if err != nil {
					log.Printf("Failed to encode discovery message: %v", err)
					continue
				}

				// Broadcast to all addresses
				for _, addr := range addresses {
					if s.broadcastConn != nil {
						_, err := s.broadcastConn.WriteToUDP(data, addr)
						if err != nil {
							// Ignore broadcast errors (some networks may block it)
							continue
						}
					}
				}
			}
			s.mu.RUnlock()
		}
	}
}
