package protocol

import "encoding/json"

// Message 是所有消息的外层包装
type Message struct {
	Type  string          `json:"type"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// RegisterRequest 注册请求（修改后：双公钥）
type RegisterRequest struct {
	Username      string `json:"username"`
	SigningKey    string `json:"signing_key"`    // Ed25519 公钥（用于签名验证）
	EncryptionKey string `json:"encryption_key"` // X25519 公钥（用于加密AES密钥）
	Algorithm     string `json:"algorithm"`      // "ed25519+x25519"
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	Type             string   `json:"type"` // "register_response"
	Success          bool     `json:"success"`
	AssignedUsername string   `json:"assigned_username"`
	OnlineUsers      []string `json:"online_users"`
}

// ChatMessage 聊天消息（修改后：双时间戳）
type ChatMessage struct {
	Type                string      `json:"type"` // "message"
	From                string      `json:"from"`
	ClientTimestamp     int64       `json:"client_timestamp"`   // 客户端时间（用于签名验证）
	ServerTimestamp     int64       `json:"server_timestamp"`   // 服务端时间（权威，用于排序）
	AESEncryptedMessage string      `json:"aes_encrypted_message"`
	Recipients          []Recipient `json:"recipients,omitempty"`    // 客户端发送时使用
	EncryptedAESKey     string      `json:"encrypted_aes_key,omitempty"` // 服务器转发时使用
	Signature           string      `json:"signature"`
}

// Recipient 接收者信息
type Recipient struct {
	To              string `json:"to"`
	EncryptedAESKey string `json:"encrypted_aes_key"`
}

// PubKeyRequest 请求公钥
type PubKeyRequest struct {
	Type  string   `json:"type"` // "get_pubkeys"
	Users []string `json:"users"`
}

// UserPublicKeys 用户公钥信息（修改后：双公钥）
type UserPublicKeys struct {
	SigningKey    string `json:"signing_key"`
	EncryptionKey string `json:"encryption_key"`
	Algorithm     string `json:"algorithm"`
}

// PubKeyResponse 公钥响应
type PubKeyResponse struct {
	Type string                    `json:"type"` // "pubkeys"
	Keys map[string]*UserPublicKeys `json:"keys"`
}

// HistoryRequest 历史消息查询请求（新增）
type HistoryRequest struct {
	Type   string `json:"type"`  // "history"
	Limit  int    `json:"limit"`
	Before int64  `json:"before,omitempty"` // 分页用，查询此时间戳之前的消息
}

// HistoryResponse 历史消息响应（新增）
type HistoryResponse struct {
	Type     string        `json:"type"` // "history_response"
	Messages []ChatMessage `json:"messages"`
	HasMore  bool          `json:"has_more"` // 是否还有更多消息
}

// UserOnlineNotification 用户上线通知（修改后：双公钥）
type UserOnlineNotification struct {
	Type          string `json:"type"` // "user_online"
	Username      string `json:"username"`
	SigningKey    string `json:"signing_key"`
	EncryptionKey string `json:"encryption_key"`
	Algorithm     string `json:"algorithm"`
}

// UserOfflineNotification 用户下线通知
type UserOfflineNotification struct {
	Type     string `json:"type"` // "user_offline"
	Username string `json:"username"`
}

// ServerShutdownNotification 服务器关闭通知（新增）
type ServerShutdownNotification struct {
	Type    string `json:"type"`    // "server_shutdown"
	Message string `json:"message"` // 关闭原因
}
