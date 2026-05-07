# go-lanlink

A lightweight LAN/WAN P2P connection tool for local multiplayer games, built with Go. Supports direct host-client connection, NAT traversal via STUN, and real-time game data forwarding.

## Features

- **Simple Room Code**: 6-digit room code for easy connection (inspired by Stardew Valley)
- **LAN Optimization**: Automatically detects and uses local network for low latency
- **NAT Traversal**: STUN-based P2P connection for cross-network play
- **Godot Integration**: Ready-to-use GDScript scripts for game developers
- **Zero Configuration**: Works out of the box, no manual IP/port input needed

## Architecture

```
┌─────────────────────────────────────────┐
│         Host's Computer                 │
│  ┌──────────────────────────────────┐  │
│  │  Godot Game (Host)               │  │
│  │  ↓ localhost:5555                │  │
│  │  lanlinkd (Go Server)            │  │
│  │  ├─ Listens on 0.0.0.0:port     │  │
│  │  ├─ STUN gets public address     │  │
│  │  └─ Generates room code          │  │
│  └──────────────────────────────────┘  │
└─────────────────────────────────────────┘
          ↑                    ↑
    LAN Direct           WAN P2P (STUN)
          |                    |
┌─────────────────┐    ┌─────────────────┐
│  Friend A (LAN) │    │  Friend B (WAN) │
└─────────────────┘    └─────────────────┘
```

## Quick Start

### For Host

1. Start lanlinkd:
```bash
go run cmd/lanlinkd/main.go host
```

2. Copy the room code (e.g., `728-491`)

3. Share with friends

### For Friends

1. Start lanlinkd:
```bash
go run cmd/lanlinkd/main.go join 728-491
```

2. Connection established automatically!

## Godot Integration

See `godot-demo/` folder for a complete example game.

Basic usage in GDScript:

```gdscript
# Host a game
var room_code = await Lanlink.create_room("Player1")
print("Room code: ", room_code)

# Join a game
await Lanlink.join_room("728-491", "Player2")

# Send game data
Lanlink.send_data({"position": player.global_position})

# Receive data
func _on_lanlink_data_received(data):
    print("Received: ", data)
```

## Protocol

The communication between Godot and lanlinkd uses JSON over UDP:

```json
// Create room
{"type": "create_room", "player_name": "Player1"}

// Room created response
{"type": "room_created", "room_code": "728-491"}

// Join room
{"type": "join_room", "room_code": "728-491", "player_name": "Player2"}

// Game data
{"type": "game_data", "payload": {"position": [100, 200]}}
```

## Development

### Project Structure

```
go-lanlink/
├── cmd/
│   └── lanlinkd/          # Main server application
├── pkg/
│   ├── server/            # Server logic
│   ├── client/            # Client logic
│   ├── protocol/          # Protocol definitions
│   └── stun/              # STUN client (future)
├── godot-demo/            # Godot example game
└── README.md
```

### Roadmap

- [x] Basic UDP server
- [x] Room code generation
- [x] Local network testing
- [ ] STUN integration
- [ ] NAT traversal
- [ ] LAN discovery
- [ ] Relay server (optional)
- [ ] Godot plugin

## License

MIT License - feel free to use in your projects!
