package crypto

import (
	"testing"
)

// TestGenerateKeyPair 测试双密钥对生成
func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// 验证签名密钥对
	if len(kp.SigningPublic) != 32 {
		t.Errorf("SigningPublic size = %d, want 32", len(kp.SigningPublic))
	}
	if len(kp.SigningPrivate) != 64 {
		t.Errorf("SigningPrivate size = %d, want 64", len(kp.SigningPrivate))
	}

	// 验证加密密钥对
	if len(kp.EncryptPublic) != 32 {
		t.Errorf("EncryptPublic size = %d, want 32", len(kp.EncryptPublic))
	}
	if len(kp.EncryptPrivate) != 32 {
		t.Errorf("EncryptPrivate size = %d, want 32", len(kp.EncryptPrivate))
	}
}

// TestX25519KeyExchange 测试 X25519 密钥交换
func TestX25519KeyExchange(t *testing.T) {
	// 生成 Alice 和 Bob 的密钥对
	alice, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Generate Alice keys failed: %v", err)
	}

	bob, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Generate Bob keys failed: %v", err)
	}

	// 测试 AES 密钥加密/解密
	testAESKey := []byte("0123456789abcdef0123456789abcdef") // 32 bytes

	// Alice 用 Bob 的公钥加密 AES 密钥
	encryptedKey, err := EncryptAESKey(testAESKey, bob.EncryptPublic, alice.EncryptPrivate)
	if err != nil {
		t.Fatalf("EncryptAESKey failed: %v", err)
	}

	// Bob 用 Alice 的公钥解密 AES 密钥
	decryptedKey, err := DecryptAESKey(encryptedKey, alice.EncryptPublic, bob.EncryptPrivate)
	if err != nil {
		t.Fatalf("DecryptAESKey failed: %v", err)
	}

	// 验证解密后的密钥是否一致
	if string(decryptedKey) != string(testAESKey) {
		t.Errorf("Decrypted key mismatch.\nGot:  %x\nWant: %x", decryptedKey, testAESKey)
	}
}

// TestSignAndVerify 测试 Ed25519 签名和验证
func TestSignAndVerify(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	testData := []byte("Hello, World!")

	// 签名
	signature := SignMessage(testData, kp.SigningPrivate)

	// 验证签名
	if !VerifySignature(testData, signature, kp.SigningPublic) {
		t.Error("Signature verification failed")
	}

	// 测试错误的签名
	wrongData := []byte("Hello, Universe!")
	if VerifySignature(wrongData, signature, kp.SigningPublic) {
		t.Error("Signature should not verify for wrong data")
	}
}

// TestEncodeDecodePublicKey 测试公钥编码/解码
func TestEncodeDecodePublicKey(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// 编码
	encoded := EncodePublicKey(kp.SigningPublic)

	// 解码
	decoded, err := DecodePublicKey(encoded)
	if err != nil {
		t.Fatalf("DecodePublicKey failed: %v", err)
	}

	// 验证
	if string(decoded) != string(kp.SigningPublic) {
		t.Error("Decoded public key mismatch")
	}
}

// TestE2EEncryption 端到端加密测试
func TestE2EEncryption(t *testing.T) {
	// 场景：Alice 发送加密消息给 Bob

	// 1. 生成密钥对
	alice, _ := GenerateKeyPair()
	bob, _ := GenerateKeyPair()

	// 2. Alice 生成 AES 密钥并加密消息
	plaintext := "Secret message from Alice to Bob"
	aesKey := make([]byte, 32)
	copy(aesKey, []byte("test-aes-key-32-bytes-long!!!!!!"))

	// 3. Alice 用 Bob 的公钥加密 AES 密钥
	encryptedAESKey, err := EncryptAESKey(aesKey, bob.EncryptPublic, alice.EncryptPrivate)
	if err != nil {
		t.Fatalf("EncryptAESKey failed: %v", err)
	}

	// 4. 模拟：Bob 收到加密的 AES 密钥，用 Alice 的公钥解密
	decryptedAESKey, err := DecryptAESKey(encryptedAESKey, alice.EncryptPublic, bob.EncryptPrivate)
	if err != nil {
		t.Fatalf("DecryptAESKey failed: %v", err)
	}

	// 5. 验证解密的 AES 密钥
	if string(decryptedAESKey) != string(aesKey) {
		t.Errorf("AES key mismatch.\nGot:  %x\nWant: %x", decryptedAESKey, aesKey)
	}

	t.Logf("✅ E2E encryption test passed: %s", plaintext)
}
