package message

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/xiaowyu/chatroom-client/pkg/crypto"
	"github.com/xiaowyu/chatroom-client/pkg/protocol"
)

// Handler 消息处理器
type Handler struct {
	keyPair     *crypto.KeyPair
	username    string
	userKeys    map[string]*protocol.UserPublicKeys // username -> 公钥
	onMessage   func(from, plaintext string, timestamp int64)
	onUserOnline  func(username string)
	onUserOffline func(username string)
}

// NewHandler 创建消息处理器
func NewHandler(kp *crypto.KeyPair, username string) *Handler {
	return &Handler{
		keyPair:  kp,
		username: username,
		userKeys: make(map[string]*protocol.UserPublicKeys),
	}
}

// SetMessageCallback 设置消息接收回调
func (h *Handler) SetMessageCallback(callback func(from, plaintext string, timestamp int64)) {
	h.onMessage = callback
}

// SetUserOnlineCallback 设置用户上线回调
func (h *Handler) SetUserOnlineCallback(callback func(username string)) {
	h.onUserOnline = callback
}

// SetUserOfflineCallback 设置用户下线回调
func (h *Handler) SetUserOfflineCallback(callback func(username string)) {
	h.onUserOffline = callback
}

// HandleMessage 处理接收到的消息
func (h *Handler) HandleMessage(msg *protocol.Message) {
	switch msg.Type {
	case "message":
		h.handleChatMessage(msg)
	case "user_online":
		h.handleUserOnline(msg)
	case "user_offline":
		h.handleUserOffline(msg)
	case "pubkeys":
		h.handlePubKeys(msg)
	case "history_response":
		h.handleHistoryResponse(msg)
	case "error":
		log.Printf("❌ Server error: %s", msg.Error)
	default:
		log.Printf("⚠️  Unknown message type: %s", msg.Type)
	}
}

// handleChatMessage 处理聊天消息
func (h *Handler) handleChatMessage(msg *protocol.Message) {
	var chatMsg protocol.ChatMessage
	if err := json.Unmarshal(msg.Data, &chatMsg); err != nil {
		log.Printf("❌ Failed to unmarshal chat message: %v", err)
		return
	}

	// 解密消息
	plaintext, err := h.decryptMessage(&chatMsg)
	if err != nil {
		log.Printf("❌ Failed to decrypt message from %s: %v", chatMsg.From, err)
		return
	}

	// 验证签名
	if !h.verifySignature(&chatMsg) {
		log.Printf("⚠️  Signature verification failed for message from %s", chatMsg.From)
		// 仍然显示消息，但添加警告标记
	}

	// 调用回调
	if h.onMessage != nil {
		h.onMessage(chatMsg.From, plaintext, chatMsg.ServerTimestamp)
	}
}

// decryptMessage 解密消息
func (h *Handler) decryptMessage(chatMsg *protocol.ChatMessage) (string, error) {
	// 1. 解密 AES 密钥（使用发送者的 X25519 公钥和自己的私钥）
	senderKeys, ok := h.userKeys[chatMsg.From]
	if !ok {
		return "", fmt.Errorf("sender public keys not found")
	}

	senderEncryptKey, err := crypto.DecodePublicKey(senderKeys.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("invalid sender encryption key: %w", err)
	}

	var senderX25519Pub [32]byte
	copy(senderX25519Pub[:], senderEncryptKey)

	// 解密 AES 密钥
	encryptedAESKey, err := crypto.DecodePublicKey(chatMsg.EncryptedAESKey)
	if err != nil {
		return "", fmt.Errorf("invalid encrypted AES key: %w", err)
	}

	aesKey, err := crypto.DecryptAESKey(encryptedAESKey, senderX25519Pub, h.keyPair.EncryptPrivate)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt AES key: %w", err)
	}

	// 2. 使用 AES 密钥解密消息
	plaintext, err := crypto.DecryptMessage(chatMsg.AESEncryptedMessage, aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt message: %w", err)
	}

	return plaintext, nil
}

// verifySignature 验证消息签名
func (h *Handler) verifySignature(chatMsg *protocol.ChatMessage) bool {
	senderKeys, ok := h.userKeys[chatMsg.From]
	if !ok {
		return false
	}

	signingKey, err := crypto.DecodePublicKey(senderKeys.SigningKey)
	if err != nil {
		return false
	}

	// 构造待签名数据（与服务端一致）
	signData := fmt.Sprintf("%s:%d:%s",
		chatMsg.From,
		chatMsg.ClientTimestamp,
		chatMsg.AESEncryptedMessage,
	)

	return crypto.VerifySignature([]byte(signData), chatMsg.Signature, signingKey)
}

// handleUserOnline 处理用户上线通知
func (h *Handler) handleUserOnline(msg *protocol.Message) {
	var notification protocol.UserOnlineNotification
	if err := json.Unmarshal(msg.Data, &notification); err != nil {
		log.Printf("❌ Failed to unmarshal user online notification: %v", err)
		return
	}

	// 保存用户公钥
	h.userKeys[notification.Username] = &protocol.UserPublicKeys{
		SigningKey:    notification.SigningKey,
		EncryptionKey: notification.EncryptionKey,
		Algorithm:     notification.Algorithm,
	}

	log.Printf("👤 User online: %s", notification.Username)

	if h.onUserOnline != nil {
		h.onUserOnline(notification.Username)
	}
}

// handleUserOffline 处理用户下线通知
func (h *Handler) handleUserOffline(msg *protocol.Message) {
	var notification protocol.UserOfflineNotification
	if err := json.Unmarshal(msg.Data, &notification); err != nil {
		log.Printf("❌ Failed to unmarshal user offline notification: %v", err)
		return
	}

	// 移除用户公钥
	delete(h.userKeys, notification.Username)

	log.Printf("👋 User offline: %s", notification.Username)

	if h.onUserOffline != nil {
		h.onUserOffline(notification.Username)
	}
}

// handlePubKeys 处理公钥响应
func (h *Handler) handlePubKeys(msg *protocol.Message) {
	var resp protocol.PubKeyResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		log.Printf("❌ Failed to unmarshal pubkeys response: %v", err)
		return
	}

	// 保存所有用户的公钥
	for username, keys := range resp.Keys {
		h.userKeys[username] = keys
	}

	log.Printf("🔑 Received public keys for %d users", len(resp.Keys))
}

// handleHistoryResponse 处理历史消息响应
func (h *Handler) handleHistoryResponse(msg *protocol.Message) {
	var resp protocol.HistoryResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		log.Printf("❌ Failed to unmarshal history response: %v", err)
		return
	}

	log.Printf("📜 Received %d history messages (hasMore: %v)", len(resp.Messages), resp.HasMore)

	// 按时间顺序解密并显示历史消息
	for i := len(resp.Messages) - 1; i >= 0; i-- {
		chatMsg := &resp.Messages[i]
		plaintext, err := h.decryptMessage(chatMsg)
		if err != nil {
			log.Printf("⚠️  Failed to decrypt history message from %s: %v", chatMsg.From, err)
			continue
		}

		if h.onMessage != nil {
			h.onMessage(chatMsg.From, plaintext, chatMsg.ServerTimestamp)
		}
	}
}

// EncryptAndSignMessage 加密并签名消息
func (h *Handler) EncryptAndSignMessage(plaintext string, recipients []string) (*protocol.ChatMessage, error) {
	// 1. 加密消息内容
	ciphertext, aesKey, err := crypto.EncryptMessage(plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt message: %w", err)
	}

	// 2. 为每个接收者加密 AES 密钥
	recipientList := make([]protocol.Recipient, 0, len(recipients))
	for _, recipientUsername := range recipients {
		recipientKeys, ok := h.userKeys[recipientUsername]
		if !ok {
			log.Printf("⚠️  Recipient %s public key not found, skipping", recipientUsername)
			continue
		}

		// 解码接收者的 X25519 公钥
		recipientEncryptKey, err := crypto.DecodePublicKey(recipientKeys.EncryptionKey)
		if err != nil {
			log.Printf("⚠️  Invalid encryption key for %s: %v", recipientUsername, err)
			continue
		}

		var recipientX25519Pub [32]byte
		copy(recipientX25519Pub[:], recipientEncryptKey)

		// 用接收者的公钥加密 AES 密钥
		encryptedAESKey, err := crypto.EncryptAESKey(aesKey, recipientX25519Pub, h.keyPair.EncryptPrivate)
		if err != nil {
			log.Printf("⚠️  Failed to encrypt AES key for %s: %v", recipientUsername, err)
			continue
		}

		recipientList = append(recipientList, protocol.Recipient{
			To:              recipientUsername,
			EncryptedAESKey: crypto.EncodePublicKey(encryptedAESKey),
		})
	}

	if len(recipientList) == 0 {
		return nil, fmt.Errorf("no valid recipients")
	}

	// 3. 签名消息
	clientTimestamp := time.Now().Unix()
	signData := fmt.Sprintf("%s:%d:%s",
		h.username,
		clientTimestamp,
		ciphertext,
	)
	signature := crypto.SignMessage([]byte(signData), h.keyPair.SigningPrivate)

	// 4. 构造消息
	chatMsg := &protocol.ChatMessage{
		Type:                "message",
		From:                h.username,
		ClientTimestamp:     clientTimestamp,
		AESEncryptedMessage: ciphertext,
		Recipients:          recipientList,
		Signature:           signature,
	}

	return chatMsg, nil
}

// GetOnlineUsers 获取在线用户列表
func (h *Handler) GetOnlineUsers() []string {
	users := make([]string, 0, len(h.userKeys))
	for username := range h.userKeys {
		if username != h.username {
			users = append(users, username)
		}
	}
	return users
}
