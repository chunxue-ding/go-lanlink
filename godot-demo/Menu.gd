extends Control

# Main menu for hosting or joining a game

signal host_pressed(player_name: String)
signal join_pressed(room_code: String, player_name: String)
signal start_game

@onready var player_name_input = $VBoxContainer/PlayerNameInput
@onready var host_button = $VBoxContainer/HostButton
@onready var join_button = $VBoxContainer/JoinButton
@onready var room_code_input = $VBoxContainer/RoomCodeInput
@onready var room_code_label = $VBoxContainer/RoomCodeLabel
@onready var start_button = $VBoxContainer/StartButton
@onready var status_label = $VBoxContainer/StatusLabel

func _ready():
	# Set default player name
	player_name_input.text = "Player" + str(randi() % 1000)

	# Connect buttons
	host_button.pressed.connect(_on_host_pressed)
	join_button.pressed.connect(_on_join_pressed)
	start_button.pressed.connect(_on_start_pressed)

	# Initial state
	room_code_label.hide()
	start_button.hide()

func _on_host_pressed():
	var player_name = player_name_input.text
	if player_name.is_empty():
		player_name = "Player"

	status_label.text = "Creating room..."
	host_pressed.emit(player_name)

func _on_join_pressed():
	var player_name = player_name_input.text
	if player_name.is_empty():
		player_name = "Player"

	var room_code = room_code_input.text
	if room_code.is_empty():
		status_label.text = "Please enter a room code"
		return

	status_label.text = "Joining room..."
	join_pressed.emit(room_code, player_name)

func set_room_code(code: String):
	room_code_label.text = "Room Code: " + code
	room_code_label.show()
	start_button.show()
	status_label.text = "Room created! Share the code with friends."

func _on_start_pressed():
	start_game.emit()
