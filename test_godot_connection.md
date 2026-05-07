# Godot 测试指南

## 前置条件

在测试之前，请确保：

1. **lanlinkd 服务器正在运行**
   ```bash
   .\bin\lanlinkd.exe host
   ```
   记下显示的房间码（例如：6294-21）

2. **Godot 4.2+ 已安装**
   - 下载：https://godotengine.org/download
   - 或者使用 Steam 版本

## 方法 1: 在 Godot Editor 中测试

### 步骤 1: 打开项目
1. 启动 Godot
2. 点击 "导入" → 浏览到 `godot-demo` 文件夹
3. 点击 "打开并编辑"

### 步骤 2: 检查项目设置
1. 项目 → 项目设置 → 应用
2. 确认主场景是 `Main.tscn`
3. 确认窗口大小设置为 1280x720

### 步骤 3: 运行项目
1. 按 **F5** 或点击 "运行项目"
2. 应该看到主菜单

### 步骤 4: 测试创建房间
1. 输入玩家名（例如："Player1"）
2. 点击 "Host Game"
3. 应该看到房间码显示（例如："Room Code: 6294-21"）
4. "Start Game" 按钮应该出现

### 步骤 5: 测试加入房间
**打开第二个 Godot 窗口：**
1. 在 Godot 编辑器中，点击 "运行当前场景的实例" (F6)
2. 或者编译导出后运行两次

**或者：**
1. 先关闭第一个窗口
2. 重新运行项目
3. 点击 "Join Game"
4. 输入房间码（例如："6294-21"）
5. 输入玩家名（例如："Player2"）

### 步骤 6: 开始游戏
1. 两个窗口都点击 "Start Game"
2. 应该看到游戏界面
3. 用 **WASD** 键移动绿色方块
4. 两个窗口应该同步显示位置

## 方法 2: 导出并运行

### 步骤 1: 导出项目
1. 在 Godot 中：项目 → 导出
2. 添加预设 → Windows/Desktop
3. 点击 "导出项目"

### 步骤 2: 运行多个实例
```powershell
# 实例 1
.\godot-demo.exe

# 实例 2（新窗口）
.\godot-demo.exe
```

## 方法 3: 使用命令行

```powershell
# 确保服务器在运行
.\bin\lanlinkd.exe host

# 运行 Godot 项目（方法 1：使用 Godot 命令）
godot.exe --path .\godot-demo

# 或（方法 2：直接运行项目）
godot.exe .\godot-demo\project.godot
```

## 预期结果

### 成功的标志：
- ✅ 主菜单正常显示
- ✅ "Host Game" 创建房间并显示房间码
- ✅ "Join Game" 成功连接
- ✅ 游戏场景显示绿色方块（玩家）
- ✅ WASD 移动流畅
- ✅ 多个窗口位置同步

### 可能的问题：

**问题 1: "Failed to connect to server"**
- 确保 `lanlinkd.exe host` 正在运行
- 检查端口 5555 是否被占用
- 查看控制台输出

**问题 2: "Room not found"**
- 检查房间码是否正确
- 确保服务器还在运行
- 尝试重新创建房间

**问题 3: 位置不同步**
- 确保两个窗口都连接到同一个房间
- 检查控制台是否有 "[DATA]" 消息
- 确认 `_process` 在发送数据

## 调试技巧

### 启用 Godot 调试输出
```gdscript
# 在 Lanlink.gd 中添加
func _receive_packet():
    var data = udp_packet.get_packet()
    print("Received: ", data.get_string_from_utf8())
```

### 查看网络流量
```gdscript
# 在 Main.gd 中添加
func _on_data_received(data: Dictionary):
    print("Data received: ", data)
```

### 检查连接状态
```gdscript
# 在 Menu.gd 中添加
func _on_host_pressed():
    print("Hosting...")
    var room_code = await Lanlink.create_room(player_name)
    print("Room created: ", room_code)
```

## 性能测试

### 测试延迟
```gdscript
# 在 Game.gd 中添加
var last_update_time = 0

func _on_data_received(data):
    var current_time = Time.get_ticks_msec()
    var latency = current_time - last_update_time
    print("Latency: ", latency, "ms")
    last_update_time = current_time
```

### 测试丢包
```gdscript
# 统计接收的包数量
var packets_received = 0
var packets_expected = 0

func _process(delta):
    packets_expected += 1

func _on_data_received(data):
    packets_received += 1
    print("Packet loss: ", (1.0 - float(packets_received) / float(packets_expected)) * 100, "%")
```

## 下一步

如果一切正常，可以：
1. 修改 `Game.gd` 实现你自己的游戏
2. 添加更多游戏状态（生命值、得分等）
3. 实现聊天系统
4. 添加玩家模型和动画
5. 优化网络性能

## 需要帮助？

如果遇到问题，请查看：
- `TEST_REPORT.md` - 命令行测试结果
- `README.md` - 项目文档
- `QUICKSTART.md` - 快速开始指南
