package user

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
)

// User 用户模型（修改后：双公钥）
type User struct {
	Username      string `json:"username"`
	SigningKey    string `json:"signing_key"`    // Ed25519 公钥 (Base64)
	EncryptionKey string `json:"encryption_key"` // X25519 公钥 (Base64)
	Algorithm     string `json:"algorithm"`      // "ed25519+x25519"
	RegisteredAt  int64  `json:"registered_at"`
}

// GetSigningKeyBytes 返回解码后的签名公钥
func (u *User) GetSigningKeyBytes() (ed25519.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(u.SigningKey)
	if err != nil {
		return nil, err
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, errors.New("invalid signing key size")
	}
	return ed25519.PublicKey(keyBytes), nil
}

// GetEncryptionKeyBytes 返回解码后的加密公钥
func (u *User) GetEncryptionKeyBytes() ([32]byte, error) {
	var key [32]byte
	keyBytes, err := base64.StdEncoding.DecodeString(u.EncryptionKey)
	if err != nil {
		return key, err
	}
	if len(keyBytes) != 32 {
		return key, errors.New("invalid encryption key size")
	}
	copy(key[:], keyBytes)
	return key, nil
}
