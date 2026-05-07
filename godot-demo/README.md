# go-lanlink Godot Demo

This is a demo Godot project showing how to integrate go-lanlink into your game.

## Prerequisites

1. **Go installed**: Download from [golang.org](https://golang.org/dl/)
2. **Godot 4.2+**: Download from [godotengine.org](https://godotengine.org/download)

## Quick Start

### Step 1: Start the lanlinkd server

Open a terminal and navigate to the go-lanlink directory:

```bash
# Host a game
go run cmd/lanlinkd/main.go host

# Or specify a custom port
go run cmd/lanlinkd/main.go host -port 6666
```

The server will display a room code (e.g., `728-491`). Share this with your friends!

### Step 2: Run the Godot demo

1. Open this project in Godot
2. Press **F5** to run
3. Enter your player name
4. Click **Host Game** (if you're the host) or **Join Game** (if you're joining)
5. For joining, enter the room code
6. Click **Start Game**

### Step 3: Test multiplayer

- Use **WASD** keys to move your character (green square)
- Open multiple instances to test with yourself
- Each player will appear as a different colored square

## Integration into Your Game

### 1. Copy the Lanlink.gd script

Copy `Lanlink.gd` to your project and add it as an autoload singleton:

```
Project Settings → AutoLoad → Add Lanlink.gd
```

### 2. Create or join a room

```gdscript
# Host a game
var room_code = await Lanlink.create_room("Player1")
print("Room code: ", room_code)

# Join a game
var result = await Lanlink.join_room("728-491", "Player2")
if result == OK:
    print("Connected!")
```

### 3. Send and receive game data

```gdscript
# Send data
func _process(delta):
    var my_data = {
        "position": [player.global_position.x, player.global_position.y],
        "rotation": player.rotation,
        "health": player.health
    }
    Lanlink.send_data(my_data)

# Receive data
func _ready():
    Lanlink.data_received.connect(_on_data_received)

func _on_data_received(data):
    player.global_position = Vector2(data.position[0], data.position[1])
    player.rotation = data.rotation
    player.health = data.health
```

### 4. Handle player events

```gdscript
func _ready():
    Lanlink.player_joined.connect(_on_player_joined)
    Lanlink.player_left.connect(_on_player_left)

func _on_player_joined(player_id, player_name):
    print(player_name, " joined the game")
    spawn_player(player_id)

func _on_player_left(player_id):
    print("Player ", player_id, " left")
    remove_player(player_id)
```

## How It Works

### Architecture

```
┌─────────────────────────────────────────┐
│         Godot Game                      │
│  ┌──────────────────────────────────┐  │
│  │  Lanlink.gd (UDP Client)        │  │
│  │  ↓ localhost:5555                │  │
│  └──────────────────────────────────┘  │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│  lanlinkd (Go Server)                   │
│  - Manages rooms                        │
│  - Forwards data between players        │
│  - Handles NAT traversal (future)       │
└─────────────────────────────────────────┘
```

### Protocol

Communication uses JSON over UDP:

```json
// Create room
{"type": "create_room", "player_name": "Player1"}

// Room created
{"type": "room_created", "room_code": "728-491"}

// Join room
{"type": "join_room", "room_code": "728-491", "player_name": "Player2"}

// Player joined
{"type": "player_joined", "player_id": "uuid", "player_name": "Player2"}

// Game data
{"type": "game_data", "payload": {"position": [100, 200]}}
```

## Limitations (Current MVP)

- **Local testing only**: Currently works on localhost only
- **No NAT traversal**: STUN coming in next phase
- **No relay**: All connections are direct
- **Basic UDP**: No reliability or ordering guarantees

## Roadmap

- [ ] LAN discovery (automatic local network detection)
- [ ] STUN integration (NAT traversal)
- [ ] UDP hole punching
- [ ] Relay server (for strict NAT)
- [ ] Data reliability (important messages)
- [ ] Lobby system

## Troubleshooting

### "Failed to connect to server"

Make sure lanlinkd is running:
```bash
go run cmd/lanlinkd/main.go host
```

### "Room not found"

Check that:
1. You entered the correct room code
2. The host's server is still running
3. Both are using the same port

### Players not syncing

Make sure you're calling `Lanlink.send_data()` in `_process()` or at regular intervals.

## Next Steps

1. **Customize the demo**: Modify `Game.gd` to implement your own game logic
2. **Add more data**: Send more complex game state (inventory, chat, etc.)
3. **Optimize**: Only send data when it changes (dirty checking)
4. **Security**: Add validation for received data

## License

MIT License - feel free to use in your projects!
