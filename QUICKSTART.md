# go-lanlink Quick Start Guide

## What is go-lanlink?

A lightweight LAN/WAN P2P connection tool for Godot games, inspired by Stardew Valley and Don't Starve. Players can connect using a simple 6-digit room code - no IP addresses or port configuration needed!

## MVP Status (Current Version)

✅ **Implemented:**
- UDP server/client in Go
- 6-digit room code generation with checksum
- JSON protocol for Godot integration
- Local multiplayer testing
- Godot demo game

⏳ **Coming Next:**
- LAN discovery (automatic local network detection)
- STUN NAT traversal (cross-network play)
- UDP hole punching
- Relay server (for strict NAT environments)

---

## Quick Test (Local)

### Step 1: Build the project

```bash
go build -o bin/lanlinkd.exe ./cmd/lanlinkd
```

Or on Linux/Mac:
```bash
go build -o bin/lanlinkd ./cmd/lanlinkd
```

### Step 2: Start a host game

Open Terminal 1:
```bash
./bin/lanlinkd.exe host
```

You'll see output like:
```
==================================================
ROOM CREATED SUCCESSFULLY!
Room Code: 728-491
Share this code with your friends!
==================================================
```

### Step 3: Join the game (simulated)

Open Terminal 2:
```bash
./bin/lanlinkd.exe join 728-491 Alice
```

You should see:
```
==================================================
CONNECTED SUCCESSFULLY!
Room: 728-491
Player ID: <uuid>
==================================================
```

### Step 4: Test data exchange

The terminal windows will show data being sent between players every 2 seconds. Try sending more data by modifying the code or integrating with Godot!

---

## Godot Integration

### Option 1: Use the demo project

```bash
cd godot-demo
# Open in Godot 4.2+ and press F5
```

### Option 2: Integrate into your game

1. **Copy `Lanlink.gd`** to your project
2. **Add it as an AutoLoad**:
   - Project Settings → AutoLoad → Browse → Select `Lanlink.gd`
   - Name it `Lanlink`

3. **Use in your code**:

```gdscript
# Host a game
func _on_host_button_pressed():
    var room_code = await Lanlink.create_room("Player1")
    $RoomCodeLabel.text = "Room: " + room_code

# Join a game
func _on_join_button_pressed(room_code: String):
    var result = await Lanlink.join_room(room_code, "Player2")
    if result == OK:
        print("Connected!")

# Send game state
func _process(delta):
    var my_data = {
        "position": [player.global_position.x, player.global_position.y],
        "velocity": [player.velocity.x, player.velocity.y]
    }
    Lanlink.send_data(my_data)

# Receive game state
func _ready():
    Lanlink.data_received.connect(_on_data_received)

func _on_data_received(data: Dictionary):
    other_player.global_position = Vector2(data.position[0], data.position[1])
```

---

## Project Structure

```
go-lanlink/
├── cmd/lanlinkd/          # Main server application
│   └── main.go            # Entry point
├── pkg/
│   ├── protocol/          # Message types and room code
│   │   ├── message.go     # JSON protocol definitions
│   │   └── roomcode.go    # Room code generation/validation
│   ├── server/            # Server logic
│   │   └── server.go      # UDP server, room management
│   └── client/            # Client logic
│       └── client.go      # UDP client, connection handling
├── godot-demo/            # Godot example project
│   ├── project.godot      # Godot project file
│   ├── Main.tscn          # Main scene
│   ├── Main.gd            # Main controller
│   ├── Lanlink.gd         # Network client (copy this!)
│   ├── Menu.gd            # Main menu
│   ├── Game.gd            # Demo game logic
│   └── README.md          # Godot-specific docs
├── bin/                   # Compiled binaries (create this folder)
├── test.bat               # Windows test script
└── README.md              # This file
```

---

## Protocol Specification

All messages are JSON over UDP (localhost:5555 by default)

### Message Types

**Create Room**
```json
{
  "type": "create_room",
  "player_name": "Player1"
}
```

**Room Created**
```json
{
  "type": "room_created",
  "room_code": "728-491"
}
```

**Join Room**
```json
{
  "type": "join_room",
  "room_code": "728-491",
  "player_name": "Player2"
}
```

**Room Joined**
```json
{
  "type": "room_joined",
  "room_code": "728-491",
  "player_id": "uuid-here"
}
```

**Player Joined**
```json
{
  "type": "player_joined",
  "player_id": "uuid-here",
  "player_name": "Player2"
}
```

**Game Data**
```json
{
  "type": "game_data",
  "payload": {
    "position": [100, 200],
    "health": 50,
    "custom_field": "value"
  }
}
```

---

## Room Code Algorithm

The 6-digit room code uses a simple checksum:

1. Generate 4 random digits
2. Calculate checksum: sum of digits % 100
3. Format: `XXXX-YY` (e.g., `728-491`)

Example:
```
Code: 7284
Sum: 7 + 2 + 8 + 4 = 21
Checksum: 21 % 100 = 21
Final: 7284-21 → Displayed as "728-421"
```

This prevents typos (e.g., `728-421` vs `728-412`)

---

## Limitations (MVP)

- **Local testing only**: Works on `localhost` only
- **No NAT traversal**: Can't connect across different networks yet
- **Basic UDP**: No reliability guarantees (packet loss possible)
- **Single server**: Each game runs its own server (no central lobby)

---

## Next Steps (Roadmap)

### Phase 1: LAN Optimization ✅ (MVP Complete)
- Basic UDP server
- Room code generation
- Local testing
- Godot integration

### Phase 2: LAN Discovery
- UDP broadcast to discover hosts on same network
- Automatic fallback to localhost
- Zero-configuration LAN play

### Phase 3: NAT Traversal
- STUN client to get public IP
- UDP hole punching
- P2P connection across internet

### Phase 4: Relay Server
- Lightweight relay for strict NAT
- Optional deployment (cloud VPS)
- Automatic fallback (P2P → relay)

### Phase 5: Production Ready
- Data reliability (ACK/retransmit)
- Connection pooling
- Error handling and recovery
- Performance optimization
- Security (encryption, authentication)

---

## Troubleshooting

### "Failed to create server"
- Make sure port 5555 is not in use
- Try: `lanlinkd host -port 6666`

### "Failed to join room"
- Verify room code is correct
- Check host server is running
- Ensure both use the same port

### Godot can't connect
- Make sure `lanlinkd` is running before starting Godot
- Check the port matches (default: 5555)
- Check firewall settings

### High latency
- MVP runs on localhost (should be < 5ms)
- If using across network, NAT traversal not implemented yet
- Stay tuned for Phase 2-3 updates!

---

## Contributing

This is an MVP project! Here's how you can help:

1. **Test locally**: Run the test script and report issues
2. **Godot integration**: Try integrating into your game
3. **Feature requests**: What do you need next?
4. **Code contributions**: PRs welcome!

---

## License

MIT License - feel free to use in your projects!

---

## Credits

Inspired by:
- **Stardew Valley** - Simple room code system
- **Don't Starve** - Host-client architecture
- **Godot Engine** - Amazing game engine

Built with:
- Go 1.21
- Godot 4.2
- Love for indie games!
