# 手动测试步骤

## 测试 1: 房间码生成和验证

在 PowerShell 中运行:
```powershell
go test ./pkg/protocol -v
```

预期输出:
```
=== RUN   TestGenerateRoomCode
--- PASS: TestGenerateRoomCode (0.00s)
PASS
```

## 测试 2: 启动服务器

终端 1:
```bash
.\bin\lanlinkd.exe host
```

预期看到:
```
Server started on port 5555
Waiting for room creation...
==================================================
ROOM CREATED SUCCESSFULLY!
Room Code: XXXX-XX
Share this code with your friends!
==================================================
```

## 测试 3: 加入房间

终端 2 (复制终端 1 显示的房间码):
```bash
.\bin\lanlinkd.exe join <房间码> TestPlayer
```

预期看到:
```
==================================================
CONNECTED SUCCESSFULLY!
Room: XXXX-XX
Player ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
==================================================
```

然后在两个终端应该看到:
```
[JOIN] TestPlayer (ID: xxx) joined the room!
[DATA] Received game data: map[...]
```

## 测试 4: Godot 集成

1. 打开 Godot 4.2+
2. 导入项目: `godot-demo/`
3. 按 F5 运行
4. 输入玩家名
5. 点击 "Host Game"
6. 复制房间码
7. 打开第二个 Godot 窗口
8. 点击 "Join Game" 并输入房间码
9. 点击 "Start Game"
10. 用 WASD 移动，应该看到两个玩家同步

## 常见问题

**Q: 提示端口被占用**
A: 使用自定义端口: `.\bin\lanlinkd.exe host -port 6666`

**Q: 加入失败**
A: 检查房间码是否正确，确保服务器正在运行

**Q: Godot 无法连接**
A: 确保先运行 `lanlinkd host`，然后再启动 Godot
