package storage

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/xiaowyu/chatroom-server/pkg/protocol"
)

// FileStorage 文件存储实现
type FileStorage struct {
	dataDir     string
	messagePath string
	userPath    string
	mu          sync.Mutex
}

// NewFileStorage 创建文件存储
func NewFileStorage(dataDir string) *FileStorage {
	return &FileStorage{
		dataDir:     dataDir,
		messagePath: filepath.Join(dataDir, "messages.jsonl"),
		userPath:    filepath.Join(dataDir, "users.json"),
	}
}

// SaveMessage 保存消息（追加到JSONL文件）
func (s *FileStorage) SaveMessage(msg *protocol.ChatMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确保目录存在
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}

	// 追加到文件
	f, err := os.OpenFile(s.messagePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = f.Write(append(data, '\n'))
	return err
}

// LoadMessages 加载消息（limit=0 表示加载全部）
func (s *FileStorage) LoadMessages(limit int) ([]*protocol.ChatMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(s.messagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*protocol.ChatMessage{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var messages []*protocol.ChatMessage
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var msg protocol.ChatMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue // 跳过损坏的行
		}
		messages = append(messages, &msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// 返回最后 limit 条
	if limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	return messages, nil
}

// SaveUsers 保存用户列表
func (s *FileStorage) SaveUsers(users []*User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.userPath, data, 0644)
}

// LoadUsers 加载用户列表
func (s *FileStorage) LoadUsers() ([]*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.userPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*User{}, nil
		}
		return nil, err
	}

	var users []*User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, err
	}

	return users, nil
}
