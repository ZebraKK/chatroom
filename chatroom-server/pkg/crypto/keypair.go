package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

// KeyPair 密钥对（修复后：双密钥对）
type KeyPair struct {
	// 签名密钥对（Ed25519）
	SigningPrivate ed25519.PrivateKey
	SigningPublic  ed25519.PublicKey

	// 加密密钥对（X25519）
	EncryptPrivate [32]byte
	EncryptPublic  [32]byte
}

// GenerateKeyPair 生成双密钥对
func GenerateKeyPair() (*KeyPair, error) {
	kp := &KeyPair{}

	// 1. 生成 Ed25519 签名密钥对
	signingPublic, signingPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	kp.SigningPublic = signingPublic
	kp.SigningPrivate = signingPrivate

	// 2. 生成 X25519 加密密钥对
	if _, err := rand.Read(kp.EncryptPrivate[:]); err != nil {
		return nil, err
	}
	// 使用 X25519 生成公钥
	publicKey, err := curve25519.X25519(kp.EncryptPrivate[:], curve25519.Basepoint)
	if err != nil {
		return nil, err
	}
	copy(kp.EncryptPublic[:], publicKey)

	return kp, nil
}

// EncryptAESKey 使用 X25519 + HKDF + ChaCha20-Poly1305 加密 AES 密钥
// 修复：Ed25519 不能用于加密，改用 X25519 ECDH
func EncryptAESKey(aesKey []byte, recipientX25519Pub [32]byte, myX25519Private [32]byte) ([]byte, error) {
	// 1. ECDH 密钥交换（使用推荐的 X25519 函数）
	sharedSecret, err := curve25519.X25519(myX25519Private[:], recipientX25519Pub[:])
	if err != nil {
		return nil, err
	}

	// 2. 使用 HKDF 派生加密密钥
	kdf := hkdf.New(sha256.New, sharedSecret[:], nil, []byte("aes-key-wrap"))
	wrapKey := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(kdf, wrapKey); err != nil {
		return nil, err
	}

	// 3. 使用 ChaCha20-Poly1305 加密 AES 密钥
	cipher, err := chacha20poly1305.New(wrapKey)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// nonce || ciphertext+tag
	ciphertext := cipher.Seal(nonce, nonce, aesKey, nil)
	return ciphertext, nil
}

// DecryptAESKey 使用 X25519 + HKDF + ChaCha20-Poly1305 解密 AES 密钥
func DecryptAESKey(encryptedAESKey []byte, senderX25519Pub [32]byte, myX25519Private [32]byte) ([]byte, error) {
	// 1. ECDH 密钥交换（使用推荐的 X25519 函数）
	sharedSecret, err := curve25519.X25519(myX25519Private[:], senderX25519Pub[:])
	if err != nil {
		return nil, err
	}

	// 2. 使用 HKDF 派生加密密钥
	kdf := hkdf.New(sha256.New, sharedSecret[:], nil, []byte("aes-key-wrap"))
	wrapKey := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(kdf, wrapKey); err != nil {
		return nil, err
	}

	// 3. 使用 ChaCha20-Poly1305 解密 AES 密钥
	cipher, err := chacha20poly1305.New(wrapKey)
	if err != nil {
		return nil, err
	}

	if len(encryptedAESKey) < cipher.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce := encryptedAESKey[:cipher.NonceSize()]
	ciphertext := encryptedAESKey[cipher.NonceSize():]

	plaintext, err := cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// SignMessage 使用 Ed25519 签名消息
func SignMessage(data []byte, signingPrivate ed25519.PrivateKey) string {
	signature := ed25519.Sign(signingPrivate, data)
	return base64.StdEncoding.EncodeToString(signature)
}

// VerifySignature 验证 Ed25519 签名
func VerifySignature(data []byte, signatureB64 string, signingPublic ed25519.PublicKey) bool {
	signature, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return false
	}
	return ed25519.Verify(signingPublic, data, signature)
}

// EncodePublicKey 编码公钥为 Base64
func EncodePublicKey(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// DecodePublicKey 解码 Base64 公钥
func DecodePublicKey(keyB64 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(keyB64)
}
