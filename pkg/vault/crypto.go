package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/nathfavour/auracrab/pkg/config"
)

// EncryptedPayload represents the structure of the encrypted secrets file
type EncryptedPayload struct {
	Version    int    `json:"version"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// getMachineID returns a unique identifier for the current machine
func getMachineID() string {
	var id string
	switch runtime.GOOS {
	case "linux":
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			id = strings.TrimSpace(string(data))
		} else if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
			id = strings.TrimSpace(string(data))
		}
	case "darwin":
		// Simplified for this context, in a real world app we might use cgo or more complex execs
		id = "mac-stable-id-fallback"
	case "windows":
		id = "win-stable-id-fallback"
	}

	if id == "" {
		// Ultimate fallback: user home directory path + hostname
		hostname, _ := os.Hostname()
		home, _ := os.UserHomeDir()
		id = hostname + ":" + home
	}

	return id
}

// deriveKey creates a 32-byte key from the machine ID and an optional salt
func deriveKey() []byte {
	id := getMachineID()
	hash := sha256.Sum256([]byte(id + "auracrab-v1-salt"))
	return hash[:]
}

// encrypt data using AES-GCM
func encrypt(data []byte) ([]byte, error) {
	key := deriveKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)

	payload := EncryptedPayload{
		Version:    1,
		Nonce:      hex.EncodeToString(nonce),
		Ciphertext: hex.EncodeToString(ciphertext),
	}

	return json.MarshalIndent(payload, "", "  ")
}

// decrypt data using AES-GCM
func decrypt(data []byte) ([]byte, error) {
	// Try to unmarshal as EncryptedPayload first
	var payload EncryptedPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		// If it's not JSON or doesn't match, it might be the old plaintext format
		// Return the original data for migration handling
		return data, fmt.Errorf("not an encrypted payload")
	}

	if payload.Version != 1 {
		return nil, fmt.Errorf("unsupported encryption version: %d", payload.Version)
	}

	key := deriveKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce, err := hex.DecodeString(payload.Nonce)
	if err != nil {
		return nil, err
	}

	ciphertext, err := hex.DecodeString(payload.Ciphertext)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// Mask returns a masked version of a secret string
func Mask(s string) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	if len(s) <= 10 {
		return s[:1] + "********" + s[len(s)-1:]
	}
	return s[:3] + "********" + s[len(s)-3:]
}

// loadSecrets reads and decrypts the secrets from disk
func loadSecrets() (map[string]string, error) {
	path := config.SecretsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}

	// Try to decrypt
	decrypted, err := decrypt(data)
	secrets := make(map[string]string)

	if err != nil {
		// Migration check: is it valid plaintext JSON?
		if err := json.Unmarshal(data, &secrets); err == nil {
			// Yes, it was plaintext. Migrate it now.
			_ = saveSecrets(secrets)
			return secrets, nil
		}
		return nil, fmt.Errorf("failed to decrypt secrets and not valid plaintext: %v", err)
	}

	if err := json.Unmarshal(decrypted, &secrets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decrypted secrets: %v", err)
	}

	return secrets, nil
}

// saveSecrets encrypts and writes the secrets to disk
func saveSecrets(secrets map[string]string) error {
	data, err := json.Marshal(secrets)
	if err != nil {
		return err
	}

	encrypted, err := encrypt(data)
	if err != nil {
		return err
	}

	return os.WriteFile(config.SecretsPath(), encrypted, 0600)
}
