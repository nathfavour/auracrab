package vault

import (
"encoding/json"
"fmt"
"os"
"sync"

"github.com/99designs/keyring"
"github.com/nathfavour/auracrab/pkg/config"
)

type Vault struct {
	ring keyring.Keyring
	mu   sync.RWMutex
}

var (
instance *Vault
once     sync.Once
)

func GetVault() *Vault {
	once.Do(func() {
		instance = &Vault{}
		ring, err := keyring.Open(keyring.Config{
ServiceName: "auracrab",
})
		if err == nil {
			instance.ring = ring
		}
	})
	return instance
}

func (v *Vault) Set(key, value string) error {
	if v.ring != nil {
		err := v.ring.Set(keyring.Item{
Key:  key,
Data: []byte(value),
})
		if err == nil {
			return nil
		}
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	secrets := make(map[string]string)
	path := config.SecretsPath()
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &secrets)
	}

	secrets[key] = value
	data, err := json.MarshalIndent(secrets, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func (v *Vault) Get(key string) (string, error) {
	if v.ring != nil {
		item, err := v.ring.Get(key)
		if err == nil {
			return string(item.Data), nil
		}
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	path := config.SecretsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	secrets := make(map[string]string)
	if err := json.Unmarshal(data, &secrets); err == nil {
		if val, ok := secrets[key]; ok {
			return val, nil
		}
	}

	return "", fmt.Errorf("secret %s not found", key)
}

func (v *Vault) List() ([]string, error) {
	if v.ring != nil {
		keys, err := v.ring.Keys()
		if err == nil {
			return keys, nil
		}
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	path := config.SecretsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	secrets := make(map[string]string)
	if err := json.Unmarshal(data, &secrets); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	return keys, nil
}
