package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/go-lanlink/pkg/client"
	"github.com/yourusername/go-lanlink/pkg/server"
	"github.com/yourusername/go-lanlink/pkg/protocol"
)

const (
	DefaultPort = 5555
)

func main() {
	// Parse command line arguments
	port := flag.Int("port", DefaultPort, "Port to listen on")
	flag.Parse()

	args := flag.Args()

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]

	switch command {
	case "host":
		runHost(*port)
	case "join":
		if len(args) < 2 {
			fmt.Println("Error: join command requires a room code")
			fmt.Println("Usage: lanlinkd join <room_code> [player_name]")
			os.Exit(1)
		}
		roomCode := args[1]
		playerName := "Player"
		if len(args) >= 3 {
			playerName = args[2]
		}
		runJoin(roomCode, playerName, *port)
	case "discover":
		playerName := "Player"
		if len(args) >= 2 {
			playerName = args[1]
		}
		runDiscover(playerName)
	case "scan":
		runScan()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("go-lanlink - LAN/WAN P2P connection tool")
	fmt.Println("\nUsage:")
	fmt.Println("  lanlinkd host [options]           - Host a game room")
	fmt.Println("  lanlinkd join <room_code> [name]  - Join a game room")
	fmt.Println("  lanlinkd discover [name]          - Discover and auto-join LAN room")
	fmt.Println("  lanlinkd scan                     - Scan for LAN rooms")
	fmt.Println("\nOptions:")
	fmt.Printf("  -port %d   - Port to use (default: %d)\n", DefaultPort, DefaultPort)
	fmt.Println("\nExamples:")
	fmt.Println("  lanlinkd host                    # Host a room")
	fmt.Println("  lanlinkd host -port 6666         # Host on custom port")
	fmt.Println("  lanlinkd join 728-491            # Join a room")
	fmt.Println("  lanlinkd join 728-491 Alice      # Join with custom name")
	fmt.Println("  lanlinkd discover                # Auto-discover and join LAN room")
	fmt.Println("  lanlinkd scan                    # List all LAN rooms")
}

func runHost(port int) {
	log.Printf("Starting host server on port %d...", port)

	// Create server
	srv, err := server.New(port)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server in background
	go srv.Start()

	log.Printf("Server started on port %d", srv.Port())
	log.Println("Waiting for room creation...")

	// Create client to communicate with server
	cli, err := client.New(client.Config{
		ServerAddr: fmt.Sprintf("localhost:%d", srv.Port()),
		PlayerName: "Host",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if err := cli.Connect(); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	// Create room
	roomCode, err := cli.CreateRoom()
	if err != nil {
		log.Fatalf("Failed to create room: %v", err)
	}

	fmt.Println("\n" + "==================================================")
	fmt.Printf("ROOM CREATED SUCCESSFULLY!\n")
	fmt.Printf("Room Code: %s\n", roomCode)
	fmt.Printf("Share this code with your friends!\n")
	fmt.Println("==================================================" + "\n")

	// Listen for incoming messages
	go func() {
		for msg := range cli.ReceiveMessage() {
			switch msg.Type {
			case protocol.TypePlayerJoined:
				fmt.Printf("[JOIN] %s (ID: %s) joined the room!\n", msg.PlayerName, msg.PlayerID)
			case protocol.TypePlayerLeft:
				fmt.Printf("[LEFT] Player %s left the room\n", msg.PlayerID)
			case protocol.TypeGameData:
				fmt.Printf("[DATA] Received game data: %v\n", msg.Payload)
			}
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("\nShutting down...")
	cli.Close()
	srv.Stop()
	fmt.Println("Goodbye!")
}

func runJoin(roomCode, playerName string, port int) {
	// Normalize room code
	roomCode = protocol.NormalizeRoomCode(roomCode)

	log.Printf("Joining room %s as %s...", roomCode, playerName)

	// Create client
	cli, err := client.New(client.Config{
		ServerAddr: fmt.Sprintf("localhost:%d", port),
		PlayerName: playerName,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if err := cli.Connect(); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	// Join room
	if err := cli.JoinRoom(roomCode); err != nil {
		log.Fatalf("Failed to join room: %v", err)
	}

	fmt.Println("\n" + "==================================================")
	fmt.Printf("CONNECTED SUCCESSFULLY!\n")
	fmt.Printf("Room: %s\n", cli.RoomCode())
	fmt.Printf("Player ID: %s\n", cli.PlayerID())
	fmt.Println("==================================================" + "\n")

	// Listen for incoming messages
	go func() {
		for msg := range cli.ReceiveMessage() {
			switch msg.Type {
			case protocol.TypePlayerJoined:
				fmt.Printf("[JOIN] %s (ID: %s) joined the room!\n", msg.PlayerName, msg.PlayerID)
			case protocol.TypePlayerLeft:
				fmt.Printf("[LEFT] Player %s left the room\n", msg.PlayerID)
			case protocol.TypeGameData:
				fmt.Printf("[DATA] Received game data: %v\n", msg.Payload)
			}
		}
	}()

	// Send test data periodically
	go func() {
		i := 0
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if cli.IsConnected() {
					data := map[string]interface{}{
						"message": fmt.Sprintf("Test from %s", playerName),
						"count":   i,
					}
					if err := cli.SendData(data); err != nil {
						log.Printf("Failed to send data: %v", err)
					}
					i++
				}
			}
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("\nShutting down...")
	cli.Close()
	fmt.Println("Goodbye!")
}

func runScan() {
	fmt.Println("Scanning for rooms on LAN...")
	fmt.Println("Timeout: 5 seconds")
	fmt.Println()

	rooms, err := client.DiscoverRooms(5 * time.Second)
	if err != nil {
		log.Fatalf("Failed to discover rooms: %v", err)
	}

	if len(rooms) == 0 {
		fmt.Println("No rooms found on LAN.")
		fmt.Println("\nTips:")
		fmt.Println("  - Make sure a host is running on the same network")
		fmt.Println("  - Check if firewall is blocking UDP port 5556")
		fmt.Println("  - Try running 'lanlinkd host' on another computer")
		return
	}

	fmt.Printf("Found %d room(s):\n", len(rooms))
	fmt.Println("==================================================")
	for i, room := range rooms {
		fmt.Printf("%d. Room: %s\n", i+1, room.RoomCode)
		fmt.Printf("   Host: %s\n", room.HostName)
		fmt.Printf("   Address: %s\n", room.Addr.IP)
		fmt.Printf("   Timestamp: %s\n", time.Unix(room.Timestamp, 0).Format("15:04:05"))
		fmt.Println()
	}
	fmt.Println("==================================================")
	fmt.Println("\nTo join a room, run:")
	fmt.Printf("  lanlinkd join %s [your_name]\n", rooms[0].RoomCode)
	fmt.Println("\nOr auto-join the first room:")
	fmt.Println("  lanlinkd discover")
}

func runDiscover(playerName string) {
	fmt.Println("Discovering and auto-joining LAN room...")
	fmt.Println("Timeout: 5 seconds")
	fmt.Println()

	cli, err := client.DiscoverAndJoin(playerName, 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to discover and join: %v", err)
	}
	defer cli.Close()

	fmt.Println("\n" + "==================================================")
	fmt.Printf("AUTO-JOINED SUCCESSFULLY!\n")
	fmt.Printf("Room: %s\n", cli.RoomCode())
	fmt.Printf("Player ID: %s\n", cli.PlayerID())
	fmt.Println("==================================================" + "\n")

	// Listen for incoming messages
	go func() {
		for msg := range cli.ReceiveMessage() {
			switch msg.Type {
			case protocol.TypePlayerJoined:
				fmt.Printf("[JOIN] %s (ID: %s) joined the room!\n", msg.PlayerName, msg.PlayerID)
			case protocol.TypePlayerLeft:
				fmt.Printf("[LEFT] Player %s left the room\n", msg.PlayerID)
			case protocol.TypeGameData:
				fmt.Printf("[DATA] Received game data: %v\n", msg.Payload)
			}
		}
	}()

	// Send test data periodically
	go func() {
		i := 0
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if cli.IsConnected() {
					data := map[string]interface{}{
						"message": fmt.Sprintf("Test from %s", playerName),
						"count":   i,
					}
					if err := cli.SendData(data); err != nil {
						log.Printf("Failed to send data: %v", err)
					}
					i++
				}
			}
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("\nShutting down...")
	cli.Close()
	fmt.Println("Goodbye!")
}
