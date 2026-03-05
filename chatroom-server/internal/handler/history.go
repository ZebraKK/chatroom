package handler

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/xiaowyu/chatroom-server/internal/connection"
	"github.com/xiaowyu/chatroom-server/internal/message"
	"github.com/xiaowyu/chatroom-server/pkg/protocol"
)

// HistoryHandler 历史消息处理器（P1功能）
type HistoryHandler struct {
	router *message.Router
}

// NewHistoryHandler 创建历史消息处理器
func NewHistoryHandler(router *message.Router) *HistoryHandler {
	return &HistoryHandler{router: router}
}

// Handle 处理历史消息查询请求
func (h *HistoryHandler) Handle(client *connection.Client, msg *protocol.Message) error {
	// 验证用户已登录
	if client.Username == "" {
		return errors.New("not authenticated")
	}

	var req protocol.HistoryRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return err
	}

	// 限制查询数量（默认20，最大100）
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// 从存储加载所有消息
	allMessages, err := h.router.GetStorage().LoadMessages(0)
	if err != nil {
		log.Printf("Failed to load messages: %v", err)
		return err
	}

	// 反向遍历 + 时间过滤 + 分页
	var filtered []protocol.ChatMessage
	for i := len(allMessages) - 1; i >= 0; i-- {
		msg := allMessages[i]

		// 如果指定了 Before 时间戳，只返回该时间之前的消息
		if req.Before > 0 && msg.ServerTimestamp >= req.Before {
			continue
		}

		// 复制消息（避免修改原始数据）
		filtered = append(filtered, *msg)

		// 达到限制数量
		if len(filtered) >= req.Limit {
			break
		}
	}

	// 检查是否还有更多消息
	hasMore := false
	if len(filtered) == req.Limit {
		// 如果返回的消息数等于请求数，可能还有更多
		hasMore = true
	}

	// 响应（包装成 Message 格式）
	resp := protocol.HistoryResponse{
		Type:     "history_response",
		Messages: filtered,
		HasMore:  hasMore,
	}

	respData, _ := json.Marshal(resp)
	envelope := protocol.Message{
		Type: "history_response",
		Data: json.RawMessage(respData),
	}
	data, _ := json.Marshal(envelope)
	client.SendChan <- data

	log.Printf("📜 History query: user=%s, limit=%d, returned=%d, hasMore=%v",
		client.Username, req.Limit, len(filtered), hasMore)

	return nil
}
