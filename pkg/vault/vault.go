package vault

import (
	"fmt"
	"sync"

	"github.com/99designs/keyring"
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

	secrets, err := loadSecrets()
	if err != nil {
		secrets = make(map[string]string)
	}

	secrets[key] = value
	return saveSecrets(secrets)
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

	secrets, err := loadSecrets()
	if err != nil {
		return "", err
	}

	if val, ok := secrets[key]; ok {
		return val, nil
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

	secrets, err := loadSecrets()
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	return keys, nil
}
