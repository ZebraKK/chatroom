package handler

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/xiaowyu/chatroom-server/internal/connection"
	"github.com/xiaowyu/chatroom-server/internal/message"
	"github.com/xiaowyu/chatroom-server/pkg/crypto"
	"github.com/xiaowyu/chatroom-server/pkg/protocol"
)

// MessageHandler 消息处理器（包含签名验证和服务端时间戳）
type MessageHandler struct {
	router *message.Router
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(router *message.Router) *MessageHandler {
	return &MessageHandler{router: router}
}

// Handle 处理消息（P0修复：实现签名验证和服务端时间戳）
func (h *MessageHandler) Handle(client *connection.Client, msg *protocol.Message) error {
	// 验证用户已登录
	if client.Username == "" {
		return errors.New("not authenticated")
	}

	// 解析消息
	var chatMsg protocol.ChatMessage
	if err := json.Unmarshal(msg.Data, &chatMsg); err != nil {
		return err
	}

	// 验证发送者
	if chatMsg.From != client.Username {
		return errors.New("invalid sender")
	}

	// ===== P0修复：签名验证 =====
	if err := h.verifyMessageSignature(&chatMsg); err != nil {
		log.Printf("❌ Signature verification failed: from=%s, error=%v", chatMsg.From, err)
		return err
	}
	log.Printf("✅ Signature verified: from=%s", chatMsg.From)

	// ===== P0修复：添加服务端权威时间戳 =====
	chatMsg.ServerTimestamp = time.Now().Unix()
	log.Printf("📅 Server timestamp added: %d (client: %d)", chatMsg.ServerTimestamp, chatMsg.ClientTimestamp)

	// 保存消息（加密状态，使用server_timestamp排序）
	if err := h.router.GetStorage().SaveMessage(&chatMsg); err != nil {
		log.Printf("Warning: failed to save message: %v", err)
	}

	// 转发给每个接收者（包含双时间戳）
	for _, recipient := range chatMsg.Recipients {
		// 检查接收者是否在线
		if !h.router.GetConnManager().IsUserOnline(recipient.To) {
			log.Printf("User %s is offline, skipping", recipient.To)
			continue
		}

		// 构造单个接收者的消息
		individualMsg := protocol.ChatMessage{
			Type:                "message",
			From:                chatMsg.From,
			ClientTimestamp:     chatMsg.ClientTimestamp,  // 客户端时间
			ServerTimestamp:     chatMsg.ServerTimestamp,  // 服务端权威时间
			AESEncryptedMessage: chatMsg.AESEncryptedMessage,
			EncryptedAESKey:     recipient.EncryptedAESKey,
			Signature:           chatMsg.Signature,
		}

		// 包装成 Message 格式
		msgData, _ := json.Marshal(individualMsg)
		envelope := protocol.Message{
			Type: "message",
			Data: json.RawMessage(msgData),
		}
		data, _ := json.Marshal(envelope)

		if err := h.router.GetConnManager().SendToUser(recipient.To, data); err != nil {
			log.Printf("Failed to send to %s: %v", recipient.To, err)
		}
	}

	log.Printf("📨 Message forwarded: from=%s, recipients=%d", chatMsg.From, len(chatMsg.Recipients))

	return nil
}

// verifyMessageSignature 验证消息签名（P0修复）
func (h *MessageHandler) verifyMessageSignature(chatMsg *protocol.ChatMessage) error {
	// 1. 获取发送者的签名公钥
	user, err := h.router.GetUserManager().GetUser(chatMsg.From)
	if err != nil {
		return fmt.Errorf("sender not found: %w", err)
	}

	signingKey, err := user.GetSigningKeyBytes()
	if err != nil {
		return fmt.Errorf("invalid signing key: %w", err)
	}

	// 2. 构造待签名数据（必须与客户端一致）
	// 格式: "from:client_timestamp:aes_encrypted_message"
	signData := fmt.Sprintf("%s:%d:%s",
		chatMsg.From,
		chatMsg.ClientTimestamp,
		chatMsg.AESEncryptedMessage,
	)

	// 3. 验证 Ed25519 签名
	if !crypto.VerifySignature([]byte(signData), chatMsg.Signature, signingKey) {
		return errors.New("signature verification failed")
	}

	// 4. 防重放攻击：验证时间戳在合理范围内（5分钟窗口）
	serverTime := time.Now().Unix()
	timeDiff := int64(math.Abs(float64(chatMsg.ClientTimestamp - serverTime)))
	if timeDiff > 300 { // 5分钟 = 300秒
		return fmt.Errorf("timestamp out of range: diff=%ds (max 300s)", timeDiff)
	}

	return nil
}

// verifySignatureDetailed 详细的签名验证（用于调试）
func (h *MessageHandler) verifySignatureDetailed(chatMsg *protocol.ChatMessage) error {
	user, err := h.router.GetUserManager().GetUser(chatMsg.From)
	if err != nil {
		return fmt.Errorf("sender not found: %w", err)
	}

	var signingKey ed25519.PublicKey
	signingKeyBytes, err := crypto.DecodePublicKey(user.SigningKey)
	if err != nil {
		return fmt.Errorf("failed to decode signing key: %w", err)
	}
	signingKey = ed25519.PublicKey(signingKeyBytes)

	signData := fmt.Sprintf("%s:%d:%s",
		chatMsg.From,
		chatMsg.ClientTimestamp,
		chatMsg.AESEncryptedMessage,
	)

	log.Printf("🔍 Signature verification details:")
	log.Printf("  From: %s", chatMsg.From)
	log.Printf("  SignData: %s", signData)
	log.Printf("  SigningKey (len): %d", len(signingKey))
	log.Printf("  Signature: %s...", chatMsg.Signature[:20])

	if !crypto.VerifySignature([]byte(signData), chatMsg.Signature, signingKey) {
		return errors.New("signature mismatch")
	}

	return nil
}
