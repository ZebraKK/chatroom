package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/xiaowyu/chatroom-client/internal/command"
	"github.com/xiaowyu/chatroom-client/internal/config"
	"github.com/xiaowyu/chatroom-client/internal/connection"
	clientCrypto "github.com/xiaowyu/chatroom-client/internal/crypto"
	"github.com/xiaowyu/chatroom-client/internal/message"
	"github.com/xiaowyu/chatroom-client/internal/ui"
	"github.com/xiaowyu/chatroom-client/pkg/crypto"
	"github.com/xiaowyu/chatroom-client/pkg/protocol"
)

func main() {
	// 命令行参数
	serverURL := flag.String("server", "", "Server WebSocket URL (wss://host:port/ws)")
	username := flag.String("username", "", "Username for registration")
	flag.Parse()

	log.SetFlags(0) // 禁用日志时间戳

	// 加载或创建配置
	cfg, isFirstRun := loadOrCreateConfig(*serverURL, *username)

	// 创建 UI
	terminal := ui.New(cfg.Username)

	// 如果是首次运行，进行初始化
	if isFirstRun {
		terminal.ShowWelcome()
		fmt.Println("🎉 Welcome! This is your first time running the chatroom client.")
		fmt.Println()

		// 生成密钥对
		fmt.Println("🔑 Generating cryptographic key pairs...")
		keyPair, err := crypto.GenerateKeyPair()
		if err != nil {
			log.Fatalf("❌ Failed to generate keys: %v", err)
		}

		// 保存密钥
		signingPath := cfg.GetSigningKeyPath()
		encryptPath := cfg.GetEncryptionKeyPath()
		if err := clientCrypto.SaveKeys(keyPair, signingPath, encryptPath); err != nil {
			log.Fatalf("❌ Failed to save keys: %v", err)
		}

		fmt.Printf("✅ Keys saved to:\n")
		fmt.Printf("   - %s\n", signingPath)
		fmt.Printf("   - %s\n", encryptPath)
		fmt.Println()

		// 保存配置
		cfg.SigningKeyPath = signingPath
		cfg.EncryptionKeyPath = encryptPath
		if err := cfg.Save(config.GetConfigPath()); err != nil {
			log.Printf("⚠️  Warning: Failed to save config: %v", err)
		}
	} else {
		terminal.ShowWelcome()
		fmt.Printf("📂 Loading configuration from %s\n", config.GetConfigPath())
	}

	// 加载密钥
	keyPair, err := clientCrypto.LoadKeys(cfg.GetSigningKeyPath(), cfg.GetEncryptionKeyPath())
	if err != nil {
		log.Fatalf("❌ Failed to load keys: %v", err)
	}
	fmt.Println("✅ Keys loaded successfully")

	// 连接到服务器
	fmt.Printf("🔌 Connecting to %s...\n", cfg.ServerURL)
	conn := connection.New(cfg.ServerURL)
	if err := conn.Connect(); err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer conn.Disconnect()

	// 创建消息处理器
	msgHandler := message.NewHandler(keyPair, cfg.Username)

	// 注册到服务器（在设置异步消息处理器之前）
	fmt.Println("📝 Registering with server...")
	registerReq := protocol.RegisterRequest{
		Username:      cfg.Username,
		SigningKey:    crypto.EncodePublicKey(keyPair.SigningPublic),
		EncryptionKey: crypto.EncodePublicKey(keyPair.EncryptPublic[:]),
		Algorithm:     "ed25519+x25519",
	}

	if err := conn.SendMessage(registerReq); err != nil {
		log.Fatalf("❌ Failed to send register request: %v", err)
	}

	// 等待注册响应
	registerResp, err := conn.ReceiveMessage()
	if err != nil {
		log.Fatalf("❌ Failed to receive register response: %v", err)
	}

	if registerResp.Type != "register_response" {
		log.Fatalf("❌ Unexpected response type: %s", registerResp.Type)
	}

	var regResp protocol.RegisterResponse
	if err := json.Unmarshal(registerResp.Data, &regResp); err != nil {
		log.Fatalf("❌ Failed to unmarshal register response: %v", err)
	}

	if !regResp.Success {
		log.Fatalf("❌ Registration failed")
	}

	// 更新用户名（可能有后缀）
	if regResp.AssignedUsername != cfg.Username {
		fmt.Printf("⚠️  Username taken, assigned: %s\n", regResp.AssignedUsername)
		cfg.Username = regResp.AssignedUsername
		msgHandler = message.NewHandler(keyPair, cfg.Username) // 使用新用户名重新创建
		terminal = ui.New(cfg.Username)                        // 重新创建 UI
	}

	// 如果有在线用户，请求他们的公钥（在设置异步处理器之前）
	if len(regResp.OnlineUsers) > 0 {
		log.Printf("📥 Requesting public keys for %d online users...", len(regResp.OnlineUsers))
		pubKeyReq := protocol.PubKeyRequest{
			Type:  "get_pubkeys",
			Users: regResp.OnlineUsers,
		}
		if err := conn.SendMessage(pubKeyReq); err != nil {
			log.Printf("⚠️  Warning: Failed to request public keys: %v", err)
		} else {
			// 等待公钥响应
			pubKeyResp, err := conn.ReceiveMessage()
			if err == nil && pubKeyResp.Type == "pubkeys" {
				// 让消息处理器处理公钥
				msgHandler.HandleMessage(pubKeyResp)
				log.Printf("✅ Received public keys for online users")
			}
		}
	}

	// 现在设置异步消息处理器（之前的同步操作已完成）
	msgHandler.SetMessageCallback(func(from, plaintext string, timestamp int64) {
		terminal.ShowMessage(from, plaintext, timestamp)
	})
	msgHandler.SetUserOnlineCallback(func(username string) {
		terminal.ShowUserOnline(username)
	})
	msgHandler.SetUserOfflineCallback(func(username string) {
		terminal.ShowUserOffline(username)
	})

	// 设置连接的消息处理器（从现在开始，所有消息都异步处理）
	conn.SetMessageHandler(msgHandler.HandleMessage)

	// 显示连接成功
	terminal.ShowConnected(regResp.OnlineUsers)

	// 创建命令处理器
	cmdHandler := command.New(conn, terminal)

	// 主循环
	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// 处理命令
		if command.IsCommand(input) {
			if shouldQuit := cmdHandler.Handle(input); shouldQuit {
				fmt.Println("👋 Goodbye!")
				break
			}
			continue
		}

		// 发送消息
		onlineUsers := msgHandler.GetOnlineUsers()
		if len(onlineUsers) == 0 {
			terminal.ShowError("No other users online")
			continue
		}

		// 加密并签名消息
		chatMsg, err := msgHandler.EncryptAndSignMessage(input, onlineUsers)
		if err != nil {
			terminal.ShowError(fmt.Sprintf("Failed to encrypt message: %v", err))
			continue
		}

		// 发送到服务器
		if err := conn.SendMessage(chatMsg); err != nil {
			terminal.ShowError(fmt.Sprintf("Failed to send message: %v", err))
			continue
		}
	}
}

// loadOrCreateConfig 加载或创建配置
func loadOrCreateConfig(serverURL, username string) (*config.Config, bool) {
	configPath := config.GetConfigPath()

	// 尝试加载现有配置
	cfg, err := config.Load(configPath)
	if err == nil {
		// 配置存在，使用现有配置
		// 但允许命令行参数覆盖
		if serverURL != "" {
			cfg.ServerURL = serverURL
		}
		if username != "" {
			cfg.Username = username
		}
		return cfg, false
	}

	// 配置不存在，创建新配置
	cfg = config.DefaultConfig()

	// 从命令行参数或交互式输入获取信息
	if serverURL != "" {
		cfg.ServerURL = serverURL
	}

	if username == "" {
		// 交互式输入用户名
		fmt.Print("Enter your username: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		username = strings.TrimSpace(input)
	}

	if username == "" {
		log.Fatal("❌ Username is required")
	}

	cfg.Username = username

	return cfg, true
}
