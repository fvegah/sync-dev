package secrets

import (
	"errors"

	"github.com/zalando/go-keyring"
)

// ServiceName is the keychain service identifier for SyncDev secrets
const ServiceName = "com.syncdev.secrets"

// ErrSecretNotFound is returned when a secret does not exist in the keychain
var ErrSecretNotFound = errors.New("secret not found")

// Manager defines the interface for managing secrets
type Manager interface {
	// GetSecret retrieves a secret for the given peer ID
	GetSecret(peerID string) (string, error)
	// SetSecret stores a secret for the given peer ID
	SetSecret(peerID, secret string) error
	// DeleteSecret removes the secret for the given peer ID
	DeleteSecret(peerID string) error
}

// KeychainManager implements Manager using the system keychain
type KeychainManager struct{}

// NewKeychainManager creates a new KeychainManager instance
func NewKeychainManager() *KeychainManager {
	return &KeychainManager{}
}

// GetSecret retrieves a secret from the keychain for the given peer ID
func (k *KeychainManager) GetSecret(peerID string) (string, error) {
	secret, err := keyring.Get(ServiceName, peerID)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrSecretNotFound
		}
		return "", err
	}
	return secret, nil
}

// SetSecret stores a secret in the keychain for the given peer ID
func (k *KeychainManager) SetSecret(peerID, secret string) error {
	return keyring.Set(ServiceName, peerID, secret)
}

// DeleteSecret removes a secret from the keychain for the given peer ID
// If the secret does not exist, no error is returned
func (k *KeychainManager) DeleteSecret(peerID string) error {
	err := keyring.Delete(ServiceName, peerID)
	if err != nil && errors.Is(err, keyring.ErrNotFound) {
		return nil // Ignore not found errors on delete
	}
	return err
}
