extends Node2D

# Simple demo game showing multiplayer sync

var players = {}
var my_position := Vector2(640, 360)

func _ready():
	# Create local player
	add_player("local", "You")

func start():
	print("Game started!")

func add_player(player_id: String, player_name: String):
	if players.has(player_id):
		return

	var player = ColorRect.new()
	player.size = Vector2(50, 50)
	player.position = Vector2(randi() % 1200, randi() % 600)

	# Random color for each player
	var color = Color(randf(), randf(), randf())
	if player_id == "local":
		color = Color.GREEN

	player.color = color
	add_child(player)

	players[player_id] = {
		"node": player,
		"name": player_name,
		"position": player.position
	}

	print("Added player: ", player_name)

func remove_player(player_id: String):
	if not players.has(player_id):
		return

	var player_data = players[player_id]
	player_data.node.queue_free()
	players.erase(player_id)

	print("Removed player: ", player_id)

func get_my_data() -> Dictionary:
	return {
		"position": [my_position.x, my_position.y],
		"color": [0, 1, 0]  # Green for local player
	}

func handle_data(data: Dictionary):
	# Handle position updates from other players
	if data.has("position"):
		var pos = data["position"]
		var position = Vector2(pos[0], pos[1])

		# Update all remote players with this position
		for player_id in players:
			if player_id != "local":
				update_player_position(player_id, position)

func update_player_position(player_id: String, position: Vector2):
	if not players.has(player_id):
		return

	players[player_id]["position"] = position
	players[player_id]["node"].position = position

func _process(delta):
	# Move local player with WASD
	var velocity = Vector2.ZERO

	if Input.is_key_pressed(KEY_W):
		velocity.y -= 1
	if Input.is_key_pressed(KEY_S):
		velocity.y += 1
	if Input.is_key_pressed(KEY_A):
		velocity.x -= 1
	if Input.is_key_pressed(KEY_D):
		velocity.x += 1

	# Normalize and apply speed
	if velocity.length() > 0:
		velocity = velocity.normalized() * 200

	my_position += velocity * delta

	# Clamp to screen
	my_position.x = clamp(my_position.x, 0, 1280 - 50)
	my_position.y = clamp(my_position.y, 0, 720 - 50)

	# Update local player position
	if players.has("local"):
		players["local"]["node"].position = my_position

func _draw():
	# Draw player names
	for player_id in players:
		var player_data = players[player_id]
		var pos = player_data["position"]
		var name = player_data["name"]

		# Draw name tag
		draw_string(ThemeDB.fallback_font, pos + Vector2(0, -10), name, HORIZONTAL_ALIGNMENT_CENTER)
