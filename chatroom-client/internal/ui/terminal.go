package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// Terminal 终端 UI
type Terminal struct {
	username    string
	onlineUsers []string
	reader      *bufio.Reader
}

// New 创建终端 UI
func New(username string) *Terminal {
	return &Terminal{
		username: username,
		reader:   bufio.NewReader(os.Stdin),
	}
}

// ShowWelcome 显示欢迎信息
func (t *Terminal) ShowWelcome() {
	t.Clear()
	fmt.Println("==============================================")
	fmt.Println("   Terminal Chatroom Client v1.0")
	fmt.Println("==============================================")
	fmt.Printf("Username:     %s\n", t.username)
	fmt.Printf("Encryption:   Ed25519+X25519 + AES-256-GCM\n")
	fmt.Println("==============================================")
	fmt.Println()
}

// ShowConnected 显示连接成功信息
func (t *Terminal) ShowConnected(onlineUsers []string) {
	t.onlineUsers = onlineUsers
	fmt.Printf("✅ Connected to server\n")
	fmt.Printf("📋 Online users (%d): %s\n", len(onlineUsers), strings.Join(onlineUsers, ", "))
	fmt.Println()
	t.showPrompt()
}

// ShowMessage 显示消息
func (t *Terminal) ShowMessage(from, message string, timestamp int64) {
	timeStr := time.Unix(timestamp, 0).Format("15:04:05")
	fmt.Printf("\r[%s] %s: %s\n", timeStr, from, message)
	t.showPrompt()
}

// ShowSystemMessage 显示系统消息
func (t *Terminal) ShowSystemMessage(message string) {
	fmt.Printf("\r💬 %s\n", message)
	t.showPrompt()
}

// ShowError 显示错误消息
func (t *Terminal) ShowError(message string) {
	fmt.Printf("\r❌ Error: %s\n", message)
	t.showPrompt()
}

// ShowUserOnline 显示用户上线
func (t *Terminal) ShowUserOnline(username string) {
	t.onlineUsers = append(t.onlineUsers, username)
	fmt.Printf("\r👤 %s joined the chat\n", username)
	t.showPrompt()
}

// ShowUserOffline 显示用户下线
func (t *Terminal) ShowUserOffline(username string) {
	// 从列表中移除
	for i, u := range t.onlineUsers {
		if u == username {
			t.onlineUsers = append(t.onlineUsers[:i], t.onlineUsers[i+1:]...)
			break
		}
	}
	fmt.Printf("\r👋 %s left the chat\n", username)
	t.showPrompt()
}

// ReadInput 读取用户输入
func (t *Terminal) ReadInput() (string, error) {
	input, err := t.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// showPrompt 显示输入提示符
func (t *Terminal) showPrompt() {
	fmt.Print("> ")
}

// Clear 清屏
func (t *Terminal) Clear() {
	fmt.Print("\033[H\033[2J")
}

// ShowHelp 显示帮助信息
func (t *Terminal) ShowHelp() {
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  /help          - Show this help message")
	fmt.Println("  /users         - List online users")
	fmt.Println("  /history [n]   - Show last n messages (default: 20)")
	fmt.Println("  /clear         - Clear screen")
	fmt.Println("  /quit          - Exit the chat")
	fmt.Println()
	fmt.Println("To send a message, just type and press Enter.")
	fmt.Println()
	t.showPrompt()
}

// ShowUsers 显示在线用户列表
func (t *Terminal) ShowUsers() {
	fmt.Println()
	fmt.Printf("📋 Online users (%d):\n", len(t.onlineUsers))
	for i, user := range t.onlineUsers {
		marker := " "
		if user == t.username {
			marker = "*"
		}
		fmt.Printf("  %s %d. %s\n", marker, i+1, user)
	}
	fmt.Println()
	t.showPrompt()
}

// GetOnlineUsers 获取在线用户列表
func (t *Terminal) GetOnlineUsers() []string {
	return t.onlineUsers
}

// SetOnlineUsers 设置在线用户列表
func (t *Terminal) SetOnlineUsers(users []string) {
	t.onlineUsers = users
}
