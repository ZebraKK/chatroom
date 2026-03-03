package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// EncryptMessage 使用 AES-256-GCM 加密消息
func EncryptMessage(plaintext string) (ciphertext string, aesKey []byte, err error) {
	// 1. 生成随机 AES-256 密钥
	aesKey = make([]byte, 32) // 256 bits
	if _, err := rand.Read(aesKey); err != nil {
		return "", nil, err
	}

	// 2. 创建 AES cipher
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", nil, err
	}

	// 3. 使用 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", nil, err
	}

	// 4. 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", nil, err
	}

	// 5. 加密
	ciphertextBytes := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	ciphertext = base64.StdEncoding.EncodeToString(ciphertextBytes)

	return ciphertext, aesKey, nil
}

// DecryptMessage 使用 AES-256-GCM 解密消息
func DecryptMessage(ciphertextB64 string, aesKey []byte) (string, error) {
	// 1. Base64 解码
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}

	// 2. 创建 AES cipher
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}

	// 3. 使用 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 4. 检查长度
	nonceSize := gcm.NonceSize()
	if len(ciphertextBytes) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// 5. 提取 nonce 和 ciphertext
	nonce, ciphertext := ciphertextBytes[:nonceSize], ciphertextBytes[nonceSize:]

	// 6. 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
