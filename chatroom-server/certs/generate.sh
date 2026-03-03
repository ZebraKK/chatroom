#!/bin/bash

# 生成自签名 TLS 证书（用于测试）

echo "Generating self-signed TLS certificate for testing..."

openssl req -x509 -newkey rsa:4096 \
    -keyout server.key \
    -out server.crt \
    -days 365 \
    -nodes \
    -subj "/CN=localhost/O=Terminal Chatroom/C=US"

chmod 600 server.key
chmod 644 server.crt

echo "✅ Certificate generated:"
echo "  - server.crt"
echo "  - server.key"
echo ""
echo "⚠️  This is a self-signed certificate for TESTING ONLY"
echo "    Do NOT use in production!"
