#!/bin/bash

# 终端聊天室快速启动脚本

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Terminal Chatroom - Quick Start"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 检查服务端是否运行
if ! pgrep -f "chatroom-server" > /dev/null; then
    echo "❌ Server is not running!"
    echo ""
    echo "Please start the server first:"
    echo "  cd /Users/xiaowyu/xwill/chatroom/chatroom-server"
    echo "  ./bin/chatroom-server"
    echo ""
    exit 1
fi

echo "✅ Server is running"
echo ""

# 检查参数
if [ -z "$1" ]; then
    echo "Usage: $0 <username>"
    echo ""
    echo "Examples:"
    echo "  Terminal 1: $0 xw"
    echo "  Terminal 2: $0 alice"
    echo "  Terminal 3: $0 bob"
    echo ""
    exit 1
fi

USERNAME=$1
CLIENT_DIR="/Users/xiaowyu/xwill/chatroom/chatroom-client"

echo "🚀 Starting client as: $USERNAME"
echo ""

cd "$CLIENT_DIR" || exit 1

if [ "$USERNAME" = "xw" ]; then
    # 默认用户，不需要 -username 参数
    ./bin/chatroom-client
else
    # 其他用户，使用 -username 参数
    ./bin/chatroom-client -username "$USERNAME"
fi
