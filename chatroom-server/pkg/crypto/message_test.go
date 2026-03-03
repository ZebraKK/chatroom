package crypto

import (
	"strings"
	"testing"
)

// TestEncryptDecryptMessage 测试消息加密和解密
func TestEncryptDecryptMessage(t *testing.T) {
	plaintext := "Hello, this is a secret message!"

	// 加密
	ciphertext, aesKey, err := EncryptMessage(plaintext)
	if err != nil {
		t.Fatalf("EncryptMessage failed: %v", err)
	}

	// 验证密文不等于明文
	if ciphertext == plaintext {
		t.Error("Ciphertext should not equal plaintext")
	}

	// 解密
	decrypted, err := DecryptMessage(ciphertext, aesKey)
	if err != nil {
		t.Fatalf("DecryptMessage failed: %v", err)
	}

	// 验证解密后的消息
	if decrypted != plaintext {
		t.Errorf("Decrypted message mismatch.\nGot:  %s\nWant: %s", decrypted, plaintext)
	}
}

// TestEncryptLongMessage 测试长消息加密
func TestEncryptLongMessage(t *testing.T) {
	// 生成一个较长的消息（1KB）
	plaintext := strings.Repeat("This is a test message. ", 50)

	ciphertext, aesKey, err := EncryptMessage(plaintext)
	if err != nil {
		t.Fatalf("EncryptMessage failed: %v", err)
	}

	decrypted, err := DecryptMessage(ciphertext, aesKey)
	if err != nil {
		t.Fatalf("DecryptMessage failed: %v", err)
	}

	if decrypted != plaintext {
		t.Error("Decrypted long message mismatch")
	}
}

// TestEncryptEmptyMessage 测试空消息加密
func TestEncryptEmptyMessage(t *testing.T) {
	plaintext := ""

	ciphertext, aesKey, err := EncryptMessage(plaintext)
	if err != nil {
		t.Fatalf("EncryptMessage failed: %v", err)
	}

	decrypted, err := DecryptMessage(ciphertext, aesKey)
	if err != nil {
		t.Fatalf("DecryptMessage failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted empty message mismatch.\nGot:  %q\nWant: %q", decrypted, plaintext)
	}
}

// TestDecryptWithWrongKey 测试用错误的密钥解密
func TestDecryptWithWrongKey(t *testing.T) {
	plaintext := "Secret message"

	ciphertext, _, err := EncryptMessage(plaintext)
	if err != nil {
		t.Fatalf("EncryptMessage failed: %v", err)
	}

	// 使用错误的密钥
	wrongKey := make([]byte, 32)
	copy(wrongKey, []byte("wrong-key-1234567890123456789!"))

	_, err = DecryptMessage(ciphertext, wrongKey)
	if err == nil {
		t.Error("DecryptMessage should fail with wrong key")
	}
}

// TestUnicodeMessage 测试 Unicode 消息
func TestUnicodeMessage(t *testing.T) {
	plaintext := "你好，世界！ 🌍 Hello, 日本語"

	ciphertext, aesKey, err := EncryptMessage(plaintext)
	if err != nil {
		t.Fatalf("EncryptMessage failed: %v", err)
	}

	decrypted, err := DecryptMessage(ciphertext, aesKey)
	if err != nil {
		t.Fatalf("DecryptMessage failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted Unicode message mismatch.\nGot:  %s\nWant: %s", decrypted, plaintext)
	}
}
