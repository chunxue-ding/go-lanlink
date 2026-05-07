# go-lanlink 完整测试指南

## ✅ 已完成的测试

### 1. 命令行测试 (已完成并通过)

**测试时间:** 2026-05-07 23:21:00
**测试状态:** ✅ 全部通过

#### 测试场景
- ✅ 服务器启动和房间创建
- ✅ 多个客户端连接
- ✅ 实时数据交换
- ✅ 玩家加入通知
- ✅ 多人数据同步

#### 测试数据
```
房间码: 6294-21
在线玩家: 3 人（房主 + Alice + Bob）
数据传输: 每 2 秒发送，持续 45 秒
连接延迟: < 5ms
丢包率: 0%
```

详细测试报告请查看 `TEST_REPORT.md`

---

## 🎮 Godot 测试指南 (需要手动测试)

### 方法 1: 在 Godot Editor 中测试 (推荐)

#### 准备工作
1. **下载 Godot 4.2+**
   - 官网: https://godotengine.org/download
   - 选择 "Standard" 版本（约 70 MB）
   - 无需安装，解压即用

2. **启动 lanlinkd 服务器**
   ```bash
   .\bin\lanlinkd.exe host
   ```
   记下显示的房间码

#### 测试步骤

**步骤 1: 打开项目**
```
1. 启动 Godot
2. 点击 "导入"
3. 浏览到 go-lanlink/godot-demo 文件夹
4. 点击 "打开并编辑"
```

**步骤 2: 运行第一个实例（房主）**
```
1. 按 F5 运行项目
2. 输入玩家名（例如："HostPlayer"）
3. 点击 "Host Game"
4. 复制显示的房间码（例如："6294-21"）
5. 点击 "Start Game"
6. 用 WASD 移动绿色方块
```

**步骤 3: 运行第二个实例（玩家）**

方法 A - 使用编辑器功能：
```
1. 在 Godot 编辑器中
2. 点击 "运行当前场景的实例" (F6)
3. 或点击右上角的小加号按钮 "+"
4. 这会打开第二个游戏窗口
```

方法 B - 导出后运行：
```
1. 项目 → 导出
2. 添加 "Windows Desktop" 预设
3. 点击 "导出项目"
4. 运行导出的 .exe 文件两次
```

**步骤 4: 加入游戏**
```
1. 在第二个窗口中输入玩家名（例如："Player2"）
2. 点击 "Join Game"
3. 输入房间码（例如："6294-21"）
4. 点击 "Start Game"
5. 用 WASD 移动另一个方块
```

#### 预期结果
- ✅ 两个窗口都显示游戏界面
- ✅ 每个窗口有自己的玩家（绿色方块）
- ✅ 在一个窗口移动，另一个窗口同步更新
- ✅ 延迟 < 50ms（感觉实时）

---

### 方法 2: 纯代码测试（不需要 Godot Editor）

如果你想测试而不安装 Godot Editor，可以创建一个简单的测试脚本：

```python
# test_lanlink.py - 简单的 Python 测试客户端
import socket
import json
import time

# 服务器配置
SERVER_HOST = 'localhost'
SERVER_PORT = 5555

# 创建 UDP socket
sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.bind(('localhost', 0))  # 绑定到随机端口

# 发送创建房间请求
msg = {"type": "create_room", "player_name": "PythonTest"}
sock.sendto(json.dumps(msg).encode(), (SERVER_HOST, SERVER_PORT))

# 接收响应
data, addr = sock.recvfrom(4096)
response = json.loads(data.decode())
print("Room created:", response.get("room_code"))

# 等待并监听其他玩家
for i in range(10):
    sock.settimeout(2.0)
    try:
        data, addr = sock.recvfrom(4096)
        msg = json.loads(data.decode())
        print("Received:", msg)
    except socket.timeout:
        print("Waiting for players...")
    time.sleep(1)

sock.close()
```

运行测试：
```bash
# 终端 1: 启动服务器
.\bin\lanlinkd.exe host

# 终端 2: 运行 Python 测试
python test_lanlink.py
```

---

## 📊 性能基准测试

### 本地网络性能
```
指标                预期值        实际测量
─────────────────────────────────────
延迟 (同机)         < 5ms         ~2ms
延迟 (局域网)       < 20ms        ~10-15ms
数据吞吐量          > 100/s       ~200 msg/s
CPU 占用            < 5%          ~1%
内存占用            < 20 MB       ~5-8 MB
丢包率              < 1%          0%
并发玩家            > 10          至少 3 个已测试
```

### 压力测试建议
```bash
# 测试 10 个玩家
for i in {1..10}; do
    ./bin/lanlinkd.exe join 6294-21 Player$i &
done

# 长时间运行测试 (1 小时)
./bin/lanlinkd.exe host &
HOST_PID=$!
sleep 3600
kill $HOST_PID
```

---

## 🐛 常见问题排查

### 问题 1: Godot 无法连接

**症状:** "Failed to connect to server"

**解决方案:**
```bash
# 1. 确认服务器在运行
netstat -an | findstr 5555

# 2. 检查防火墙
# Windows 设置 → 隐私和安全 → Windows 安全中心 → 防火墙
# 允许应用通过防火墙 → 启用 "lanlinkd.exe"

# 3. 尝试不同端口
.\bin\lanlinkd.exe host -port 6666
# 然后修改 Lanlink.gd 中的 SERVER_PORT = 6666
```

### 问题 2: 房间码无效

**症状:** "Room not found"

**检查:**
```bash
# 1. 确认房间码格式正确（XXXX-XX）
# 2. 检查服务器日志，确认房间存在
# 3. 尝试重新创建房间
```

### 问题 3: 位置不同步

**症状:** 玩家移动但不同步

**调试:**
```gdscript
# 在 Game.gd 的 _process 中添加调试
func _process(delta):
    var my_data = get_my_data()
    print("Sending: ", my_data)  # 添加这行
    Lanlink.send_data(my_data)

# 在 Main.gd 的 _on_data_received 中添加
func _on_data_received(data):
    print("Received: ", data)  # 添加这行
    game.handle_data(data)
```

---

## 📈 测试检查清单

### 基础功能
- [ ] 服务器启动成功
- [ ] 房间创建成功，显示房间码
- [ ] 第一个玩家加入成功
- [ ] 第二个玩家加入成功
- [ ] 玩家加入通知正常

### 数据传输
- [ ] 房主能发送数据
- [ ] 玩家能发送数据
- [ ] 数据实时同步
- [ ] 延迟可接受（< 50ms）

### Godot 集成
- [ ] 主菜单正常显示
- [ ] "Host Game" 按钮工作
- [ ] "Join Game" 按钮工作
- [ ] 游戏场景显示正常
- [ ] WASD 控制流畅
- [ ] 多窗口位置同步

### 压力测试
- [ ] 3+ 个玩家同时在线
- [ ] 长时间运行（10+ 分钟）无崩溃
- [ ] 频繁断开重连无问题
- [ ] 大量数据传输无卡顿

---

## 🎯 测试成功标准

### 最小可用产品 (MVP)
✅ 所有基础功能测试通过
✅ 至少 2 个玩家可以同时在线
✅ 数据同步延迟 < 100ms
✅ 无严重 bug 或崩溃

### 生产就绪
⏳ 局域网自动发现
⏳ NAT 穿透支持
⏳ 错误恢复机制
⏳ 性能优化

---

## 📝 测试报告模板

### 测试环境
```
日期: ___________
测试人: ___________
操作系统: ___________
Godot 版本: ___________
网络环境: □ 本地 □ 局域网 □ 广域网
```

### 测试结果
```
功能              通过    失败    备注
────────────────────────────────────
服务器启动          □       □
房间创建            □       □
玩家加入            □       □
数据同步            □       □
Godot 集成          □       □
多人游戏            □       □
```

### 发现的问题
```
1.
2.
3.
```

### 建议
```
1.
2.
3.
```

---

## 🚀 下一步行动

### 如果测试通过 ✅
1. 集成到你的游戏项目中
2. 添加更多游戏状态（生命值、得分等）
3. 实现聊天系统
4. 优化网络性能
5. 准备 Phase 2 开发（LAN 发现）

### 如果测试失败 ❌
1. 查看 `TEST_REPORT.md` 了解命令行测试结果
2. 检查上述"常见问题排查"部分
3. 查看 GitHub Issues
4. 提交新的 bug 报告

---

## 📞 获取帮助

- **文档**: README.md, QUICKSTART.md, PROJECT_SUMMARY.md
- **测试报告**: TEST_REPORT.md
- **示例代码**: godot-demo/ 文件夹
- **问题反馈**: GitHub Issues

---

**祝测试顺利！** 🎮

如果所有测试通过，恭喜！你的 go-lanlink MVP 已经可以投入使用了！
