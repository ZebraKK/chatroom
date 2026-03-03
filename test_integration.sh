#!/bin/bash

# 终端聊天室集成测试脚本

echo "=========================================="
echo "  Terminal Chatroom - Integration Test"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试结果
PASSED=0
FAILED=0

# 测试函数
test_pass() {
    echo -e "${GREEN}✅ PASS${NC}: $1"
    ((PASSED++))
}

test_fail() {
    echo -e "${RED}❌ FAIL${NC}: $1"
    ((FAILED++))
}

test_info() {
    echo -e "${YELLOW}ℹ️  INFO${NC}: $1"
}

# 1. 检查编译产物
echo "1️⃣  Checking build artifacts..."
if [ -f "chatroom-server/bin/chatroom-server" ]; then
    test_pass "Server binary exists"
else
    test_fail "Server binary not found"
fi

if [ -f "chatroom-client/bin/chatroom-client" ]; then
    test_pass "Client binary exists"
else
    test_fail "Client binary not found"
fi

# 2. 检查证书
echo ""
echo "2️⃣  Checking TLS certificates..."
if [ -f "chatroom-server/certs/server.crt" ]; then
    test_pass "TLS certificate exists"
else
    test_fail "TLS certificate not found"
fi

if [ -f "chatroom-server/certs/server.key" ]; then
    test_pass "TLS key exists"
else
    test_fail "TLS key not found"
fi

# 3. 启动服务端
echo ""
echo "3️⃣  Starting server..."
cd chatroom-server
./bin/chatroom-server &
SERVER_PID=$!
cd ..

sleep 2

if ps -p $SERVER_PID > /dev/null; then
    test_pass "Server started successfully (PID: $SERVER_PID)"
else
    test_fail "Server failed to start"
    exit 1
fi

# 4. 测试总结
echo ""
echo "=========================================="
echo "  Test Summary"
echo "=========================================="
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All checks passed!${NC}"
    echo ""
    echo "Server is running on https://localhost:8443"
    echo "Server PID: $SERVER_PID"
    echo ""
    echo "To test the client, run in another terminal:"
    echo "  cd chatroom-client"
    echo "  ./bin/chatroom-client"
    echo ""
    echo "To stop the server:"
    echo "  kill $SERVER_PID"
else
    echo -e "${RED}❌ Some tests failed${NC}"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi
