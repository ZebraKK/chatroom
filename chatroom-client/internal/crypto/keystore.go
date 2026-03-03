package crypto

import (
	"crypto/ed25519"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/xiaowyu/chatroom-client/pkg/crypto"
)

// KeyStore 密钥存储结构
type KeyStore struct {
	SigningPrivate    []byte `json:"signing_private"`     // Ed25519 私钥
	SigningPublic     []byte `json:"signing_public"`      // Ed25519 公钥
	EncryptionPrivate []byte `json:"encryption_private"`  // X25519 私钥
	EncryptionPublic  []byte `json:"encryption_public"`   // X25519 公钥
	Algorithm         string `json:"algorithm"`           // "ed25519+x25519"
}

// SaveKeys 保存密钥对到文件
func SaveKeys(kp *crypto.KeyPair, signingPath, encryptPath string) error {
	// 确保目录存在
	dir := filepath.Dir(signingPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// 保存签名密钥
	signingStore := &KeyStore{
		SigningPrivate: kp.SigningPrivate,
		SigningPublic:  kp.SigningPublic,
		Algorithm:      "ed25519",
	}
	signingData, err := json.MarshalIndent(signingStore, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(signingPath, signingData, 0600); err != nil {
		return err
	}

	// 保存加密密钥
	encryptStore := &KeyStore{
		EncryptionPrivate: kp.EncryptPrivate[:],
		EncryptionPublic:  kp.EncryptPublic[:],
		Algorithm:         "x25519",
	}
	encryptData, err := json.MarshalIndent(encryptStore, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(encryptPath, encryptData, 0600); err != nil {
		return err
	}

	return nil
}

// LoadKeys 从文件加载密钥对
func LoadKeys(signingPath, encryptPath string) (*crypto.KeyPair, error) {
	kp := &crypto.KeyPair{}

	// 加载签名密钥
	signingData, err := os.ReadFile(signingPath)
	if err != nil {
		return nil, err
	}
	var signingStore KeyStore
	if err := json.Unmarshal(signingData, &signingStore); err != nil {
		return nil, err
	}
	kp.SigningPrivate = ed25519.PrivateKey(signingStore.SigningPrivate)
	kp.SigningPublic = ed25519.PublicKey(signingStore.SigningPublic)

	// 加载加密密钥
	encryptData, err := os.ReadFile(encryptPath)
	if err != nil {
		return nil, err
	}
	var encryptStore KeyStore
	if err := json.Unmarshal(encryptData, &encryptStore); err != nil {
		return nil, err
	}
	copy(kp.EncryptPrivate[:], encryptStore.EncryptionPrivate)
	copy(kp.EncryptPublic[:], encryptStore.EncryptionPublic)

	return kp, nil
}

// KeysExist 检查密钥文件是否存在
func KeysExist(signingPath, encryptPath string) bool {
	_, err1 := os.Stat(signingPath)
	_, err2 := os.Stat(encryptPath)
	return err1 == nil && err2 == nil
}
