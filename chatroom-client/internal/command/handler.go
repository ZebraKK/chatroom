package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xiaowyu/chatroom-client/internal/connection"
	"github.com/xiaowyu/chatroom-client/internal/ui"
	"github.com/xiaowyu/chatroom-client/pkg/protocol"
)

// Handler 命令处理器
type Handler struct {
	conn *connection.Connection
	ui   *ui.Terminal
}

// New 创建命令处理器
func New(conn *connection.Connection, ui *ui.Terminal) *Handler {
	return &Handler{
		conn: conn,
		ui:   ui,
	}
}

// Handle 处理命令
// 返回 true 表示应该退出
func (h *Handler) Handle(input string) bool {
	if !strings.HasPrefix(input, "/") {
		return false
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/help":
		h.ui.ShowHelp()
	case "/users":
		h.ui.ShowUsers()
	case "/history":
		h.handleHistory(args)
	case "/clear":
		h.ui.Clear()
		h.ui.ShowWelcome()
	case "/quit", "/exit":
		return true
	default:
		h.ui.ShowError(fmt.Sprintf("Unknown command: %s (type /help for help)", cmd))
	}

	return false
}

// handleHistory 处理历史消息命令
func (h *Handler) handleHistory(args []string) {
	limit := 20
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
			if n > 100 {
				limit = 100
			} else {
				limit = n
			}
		}
	}

	// 发送历史消息请求
	req := protocol.HistoryRequest{
		Type:  "history",
		Limit: limit,
	}

	if err := h.conn.SendMessage(req); err != nil {
		h.ui.ShowError(fmt.Sprintf("Failed to request history: %v", err))
		return
	}

	h.ui.ShowSystemMessage(fmt.Sprintf("Loading last %d messages...", limit))
}

// IsCommand 检查输入是否是命令
func IsCommand(input string) bool {
	return strings.HasPrefix(input, "/")
}
