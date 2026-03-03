package storage

import (
	"github.com/xiaowyu/chatroom-server/pkg/protocol"
)

// User 用户数据（避免循环依赖）
type User struct {
	Username      string `json:"username"`
	SigningKey    string `json:"signing_key"`
	EncryptionKey string `json:"encryption_key"`
	Algorithm     string `json:"algorithm"`
	RegisteredAt  int64  `json:"registered_at"`
}

// Storage 存储接口
type Storage interface {
	// 消息相关
	SaveMessage(msg *protocol.ChatMessage) error
	LoadMessages(limit int) ([]*protocol.ChatMessage, error)

	// 用户相关
	SaveUsers(users []*User) error
	LoadUsers() ([]*User, error)
}
