extends Node

# Lanlink client for Godot
# Handles communication with lanlinkd server

signal player_joined(player_id: String, player_name: String)
signal player_left(player_id: String)
signal data_received(data: Dictionary)
signal connected()
signal connection_failed(error: String)

const SERVER_PORT = 5555
const BUFFER_SIZE = 4096

var udp_packet := PacketPeerUDP.new()
var connected_players := {}
var my_player_id := ""
var my_room_code := ""
var is_connected := false

func _ready():
	set_process(true)

func _process(delta):
	# Check for incoming packets
	if udp_packet.get_available_packet_count() > 0:
		_receive_packet()

func create_room(player_name: String) -> String:
	# Send create room request
	var msg = {
		"type": "create_room",
		"player_name": player_name
	}
	_send_message(msg)

	# Wait for response
	var response = await _wait_for_message("room_created", 5.0)
	if response.is_empty():
		connection_failed.emit("Timeout waiting for room creation")
		return ""

	my_room_code = response.get("room_code", "")
	is_connected = true
	connected.emit()
	return my_room_code

func join_room(room_code: String, player_name: String) -> int:
	# Send join room request
	var msg = {
		"type": "join_room",
		"room_code": room_code,
		"player_name": player_name
	}
	_send_message(msg)

	# Wait for response
	var response = await _wait_for_message("room_joined", 5.0)
	if response.is_empty():
		var err_response = await _wait_for_message("room_not_found", 1.0)
		if not err_response.is_empty():
			connection_failed.emit("Room not found: " + room_code)
			return ERR_CANT_CONNECT
		connection_failed.emit("Timeout joining room")
		return ERR_CANT_CONNECT

	my_room_code = response.get("room_code", "")
	my_player_id = response.get("player_id", "")
	is_connected = true
	connected.emit()
	return OK

func send_data(data: Dictionary):
	if not is_connected:
		return

	var msg = {
		"type": "game_data",
		"payload": data
	}
	_send_message(msg)

func _send_message(msg: Dictionary):
	var json = JSON.stringify(msg)
	var packet = json.to_utf8_buffer()

	if udp_packet.put_packet(packet) != OK:
		print("Failed to send packet")

func _receive_packet():
	var data = udp_packet.get_packet()
	if data.is_empty():
		return

	var json_string = data.get_string_from_utf8()
	var json = JSON.new()
	var parse_result = json.parse(json_string)

	if parse_result != OK:
		print("Failed to parse JSON: ", json_string)
		return

	var msg = json.data
	if not msg is Dictionary:
		return

	var type = msg.get("type", "")
	match type:
		"room_created":
			# Will be handled by await
			pass
		"room_joined":
			my_player_id = msg.get("player_id", "")
		"room_not_found":
			pass
		"player_joined":
			var player_id = msg.get("player_id", "")
			var player_name = msg.get("player_name", "")
			connected_players[player_id] = player_name
			player_joined.emit(player_id, player_name)
		"player_left":
			var player_id = msg.get("player_id", "")
			if connected_players.has(player_id):
				connected_players.erase(player_id)
			player_left.emit(player_id)
		"game_data":
			var payload = msg.get("payload", {})
			data_received.emit(payload)
		"error":
			var error_msg = msg.get("error", "Unknown error")
			print("Server error: ", error_msg)

func _wait_for_message(msg_type: String, timeout: float) -> Dictionary:
	var start_time = Time.get_time_dict_from_system()

	while Time.get_time_dict_from_system() - start_time < timeout:
		if udp_packet.get_available_packet_count() > 0:
			var data = udp_packet.get_packet()
			var json_string = data.get_string_from_utf8()
			var json = JSON.new()
			if json.parse(json_string) == OK:
				var msg = json.data
				if msg is Dictionary and msg.get("type", "") == msg_type:
					return msg

		await get_tree().process_frame

	return {}

func is_room_connected() -> bool:
	return is_connected
