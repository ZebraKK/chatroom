package main

import (
	"flag"
	"log"

	"github.com/xiaowyu/chatroom-server/internal/server"
)

func main() {
	// 命令行参数
	addr := flag.String("addr", ":8443", "Server address")
	certFile := flag.String("cert", "./certs/server.crt", "TLS certificate file")
	keyFile := flag.String("key", "./certs/server.key", "TLS key file")
	dataDir := flag.String("data", "./data", "Data directory")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("==============================================")
	log.Println("   Terminal Chatroom Server v1.0")
	log.Println("==============================================")
	log.Printf("Address:      %s", *addr)
	log.Printf("Certificate:  %s", *certFile)
	log.Printf("Key:          %s", *keyFile)
	log.Printf("Data Dir:     %s", *dataDir)
	log.Println("==============================================")

	// 创建并启动服务器
	srv := server.New(*addr, *certFile, *keyFile, *dataDir)
	if err := srv.Start(); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
