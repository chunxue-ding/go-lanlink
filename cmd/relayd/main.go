package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/go-lanlink/pkg/relay"
)

const (
	DefaultPort = 7000
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
	case "start":
		runRelay(*port)
	case "stats":
		showStats(*port)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("relayd - go-lanlink relay server")
	fmt.Println("\nUsage:")
	fmt.Println("  relayd start [options]  - Start relay server")
	fmt.Println("  relayd stats           - Show relay statistics")
	fmt.Println("\nOptions:")
	fmt.Printf("  -port %d   - Port to listen on (default: %d)\n", DefaultPort, DefaultPort)
	fmt.Println("\nExamples:")
	fmt.Println("  relayd start                  # Start on default port 7000")
	fmt.Println("  relayd start -port 8000       # Start on custom port")
	fmt.Println("\nDeployment:")
	fmt.Println("  # Direct")
	fmt.Println("  ./relayd start")
	fmt.Println("\n  # Docker")
	fmt.Println("  docker run -p 7000:7000 ghcr.io/yourusername/relayd:latest")
	fmt.Println("\n  # Systemd service")
	fmt.Println("  sudo cp relayd.service /etc/systemd/system/")
	fmt.Println("  sudo systemctl start relayd")
}

func runRelay(port int) {
	log.Printf("Starting relay server on port %d...", port)

	// Create relay server
	config := relay.DefaultConfig()
	config.Port = port

	srv, err := relay.New(config)
	if err != nil {
		log.Fatalf("Failed to create relay server: %v", err)
	}

	// Start server in background
	go srv.Start()

	log.Printf("Relay server is ready!")
	log.Printf("Listening on port %d", port)
	log.Printf("Players can connect using:")
	log.Printf("  lanlinkd join <room_code> <name> --relay localhost:%d", port)
	log.Println("")
	log.Println("Press Ctrl+C to stop")

	// Print stats periodically
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			stats := srv.GetStats()
			log.Printf("[STATS] Sessions: %d, Clients: %d, Forwarded: %d packets (%d MB)",
				stats.TotalSessions,
				stats.TotalClients,
				stats.PacketsForwarded,
				stats.BytesForwarded/(1024*1024))
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("\nShutting down relay server...")
	srv.Stop()

	// Print final stats
	stats := srv.GetStats()
	fmt.Println("\n" + "==================================================")
	fmt.Println("Final Statistics:")
	fmt.Printf("  Total Sessions: %d\n", stats.TotalSessions)
	fmt.Printf("  Total Clients: %d\n", stats.TotalClients)
	fmt.Printf("  Packets Forwarded: %d\n", stats.PacketsForwarded)
	fmt.Printf("  Bytes Forwarded: %.2f MB\n", float64(stats.BytesForwarded)/(1024*1024))
	fmt.Println("==================================================")

	fmt.Println("Relay server stopped. Goodbye!")
}

func showStats(port int) {
	// TODO: Implement stats query via admin API
	fmt.Println("Stats query not yet implemented")
	fmt.Println("Please check the server logs for statistics")
}
