package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Config 客户端配置
type Config struct {
	Username      string `json:"username"`
	ServerURL     string `json:"server_url"`
	Algorithm     string `json:"algorithm"`
	KeyPath       string `json:"key_path"`
	SigningKeyPath    string `json:"signing_key_path"`
	EncryptionKeyPath string `json:"encryption_key_path"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		ServerURL: "wss://localhost:8443/ws",
		Algorithm: "ed25519+x25519",
		KeyPath:   filepath.Join(homeDir, ".chatroom", "keys"),
	}
}

// Load 加载配置文件
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("config file not found")
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save 保存配置文件
func (c *Config) Save(path string) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".chatroom", "config.json")
}

// GetKeysDir 获取密钥目录
func (c *Config) GetKeysDir() string {
	if c.KeyPath != "" {
		return c.KeyPath
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".chatroom", "keys")
}

// GetSigningKeyPath 获取签名密钥路径
func (c *Config) GetSigningKeyPath() string {
	if c.SigningKeyPath != "" {
		return c.SigningKeyPath
	}
	return filepath.Join(c.GetKeysDir(), c.Username+"_signing.key")
}

// GetEncryptionKeyPath 获取加密密钥路径
func (c *Config) GetEncryptionKeyPath() string {
	if c.EncryptionKeyPath != "" {
		return c.EncryptionKeyPath
	}
	return filepath.Join(c.GetKeysDir(), c.Username+"_encrypt.key")
}
