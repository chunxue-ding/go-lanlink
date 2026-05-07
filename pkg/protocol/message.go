package protocol

import "encoding/json"

// Message types
const (
	TypeCreateRoom   = "create_room"
	TypeRoomCreated  = "room_created"
	TypeJoinRoom     = "join_room"
	TypeRoomJoined   = "room_joined"
	TypeRoomNotFound = "room_not_found"
	TypeGameData     = "game_data"
	TypePlayerJoined = "player_joined"
	TypePlayerLeft   = "player_left"
	TypeError        = "error"
)

// Message represents a communication message between Godot and lanlinkd
type Message struct {
	Type           string                 `json:"type"`
	PlayerName     string                 `json:"player_name,omitempty"`
	RoomCode       string                 `json:"room_code,omitempty"`
	Payload        map[string]interface{} `json:"payload,omitempty"`
	Error          string                 `json:"error,omitempty"`
	PlayerID       string                 `json:"player_id,omitempty"`
	PublicIP       string                 `json:"public_ip,omitempty"`
	PublicPort     int                    `json:"public_port,omitempty"`
	PublicAddress  string                 `json:"public_address,omitempty"` // "ip:port" format
}

// Encode converts a Message to JSON bytes
func (m *Message) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// Decode parses JSON bytes into a Message
func Decode(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// Helper functions to create common messages
func NewCreateRoomMessage(playerName string) *Message {
	return &Message{
		Type:       TypeCreateRoom,
		PlayerName: playerName,
	}
}

func NewRoomCreatedMessage(roomCode string) *Message {
	return &Message{
		Type:     TypeRoomCreated,
		RoomCode: roomCode,
	}
}

func NewJoinRoomMessage(roomCode, playerName string) *Message {
	return &Message{
		Type:       TypeJoinRoom,
		RoomCode:   roomCode,
		PlayerName: playerName,
	}
}

func NewRoomJoinedMessage(roomCode, playerID string) *Message {
	return &Message{
		Type:     TypeRoomJoined,
		RoomCode: roomCode,
		PlayerID: playerID,
	}
}

func NewRoomNotFoundMessage(roomCode string) *Message {
	return &Message{
		Type:     TypeRoomNotFound,
		RoomCode: roomCode,
	}
}

func NewGameDataMessage(payload map[string]interface{}) *Message {
	return &Message{
		Type:    TypeGameData,
		Payload: payload,
	}
}

func NewPlayerJoinedMessage(playerID, playerName string) *Message {
	return &Message{
		Type:       TypePlayerJoined,
		PlayerID:   playerID,
		PlayerName: playerName,
	}
}

func NewPlayerLeftMessage(playerID string) *Message {
	return &Message{
		Type:     TypePlayerLeft,
		PlayerID: playerID,
	}
}

func NewErrorMessage(err string) *Message {
	return &Message{
		Type:  TypeError,
		Error: err,
	}
}
