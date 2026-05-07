extends Node

# Main scene for go-lanlink demo

@onready var menu = $Menu
@onready var game = $Game
@onready var lanlink = $Lanlink

func _ready():
	# Connect Lanlink signals
	lanlink.player_joined.connect(_on_player_joined)
	lanlink.player_left.connect(_on_player_left)
	lanlink.data_received.connect(_on_data_received)

	# Show menu
	menu.show()
	game.hide()

	# Connect menu buttons
	menu.host_pressed.connect(_on_host_pressed)
	menu.join_pressed.connect(_on_join_pressed)

func _on_host_pressed(player_name: String):
	# Create a room
	var room_code = await lanlink.create_room(player_name)
	print("Room created: ", room_code)
	menu.set_room_code(room_code)

	# Wait for player to start game
	await menu.start_game

	# Switch to game scene
	_start_game()

func _on_join_pressed(room_code: String, player_name: String):
	# Join a room
	var result = await lanlink.join_room(room_code, player_name)
	if result == OK:
		print("Joined room: ", room_code)
		# Wait a bit for other players
		await get_tree().create_timer(1.0).timeout
		_start_game()
	else:
		print("Failed to join room")

func _start_game():
	menu.hide()
	game.show()
	game.start()

func _on_player_joined(player_id: String, player_name: String):
	print("Player joined: ", player_name)
	game.add_player(player_id, player_name)

func _on_player_left(player_id: String):
	print("Player left: ", player_id)
	game.remove_player(player_id)

func _on_data_received(data: Dictionary):
	print("Data received: ", data)
	game.handle_data(data)

func _process(delta):
	# Send game data
	if game.visible:
		var my_data = game.get_my_data()
		lanlink.send_data(my_data)
