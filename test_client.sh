#!/bin/bash

# 自动化客户端测试脚本

USERNAME=$1
if [ -z "$USERNAME" ]; then
    echo "Usage: $0 <username>"
    exit 1
fi

echo "Testing client with username: $USERNAME"

cd chatroom-client

# 创建测试输入
TEST_INPUT=$(cat <<EOF
你好，我是 $USERNAME
/users
/help
/quit
EOF
)

# 运行客户端
echo "$TEST_INPUT" | ./bin/chatroom-client -username "$USERNAME" -server "wss://localhost:8443/ws"
