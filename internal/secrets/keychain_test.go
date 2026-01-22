package secrets

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	// Use mock keyring for all tests
	keyring.MockInit()
	m.Run()
}

func TestSetAndGetSecret(t *testing.T) {
	km := NewKeychainManager()
	peerID := "test-peer-123"
	secretValue := "my-super-secret-value"

	// Set the secret
	err := km.SetSecret(peerID, secretValue)
	if err != nil {
		t.Fatalf("SetSecret failed: %v", err)
	}

	// Get the secret back
	retrieved, err := km.GetSecret(peerID)
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}

	if retrieved != secretValue {
		t.Errorf("Expected secret '%s', got '%s'", secretValue, retrieved)
	}
}

func TestGetSecretNotFound(t *testing.T) {
	km := NewKeychainManager()
	peerID := "nonexistent-peer"

	_, err := km.GetSecret(peerID)
	if err == nil {
		t.Error("Expected error for nonexistent secret, got nil")
	}

	if err != ErrSecretNotFound {
		t.Errorf("Expected ErrSecretNotFound, got: %v", err)
	}
}

func TestDeleteSecret(t *testing.T) {
	km := NewKeychainManager()
	peerID := "peer-to-delete"
	secretValue := "secret-to-be-deleted"

	// First set a secret
	err := km.SetSecret(peerID, secretValue)
	if err != nil {
		t.Fatalf("SetSecret failed: %v", err)
	}

	// Verify it exists
	_, err = km.GetSecret(peerID)
	if err != nil {
		t.Fatalf("GetSecret failed after set: %v", err)
	}

	// Delete it
	err = km.DeleteSecret(peerID)
	if err != nil {
		t.Fatalf("DeleteSecret failed: %v", err)
	}

	// Verify it's gone
	_, err = km.GetSecret(peerID)
	if err != ErrSecretNotFound {
		t.Errorf("Expected ErrSecretNotFound after delete, got: %v", err)
	}
}

func TestDeleteSecretNonexistent(t *testing.T) {
	km := NewKeychainManager()
	peerID := "never-existed-peer"

	// Delete nonexistent secret should not error
	err := km.DeleteSecret(peerID)
	if err != nil {
		t.Errorf("DeleteSecret on nonexistent should not error, got: %v", err)
	}
}

func TestUpdateSecret(t *testing.T) {
	km := NewKeychainManager()
	peerID := "updatable-peer"
	originalSecret := "original-secret"
	updatedSecret := "updated-secret"

	// Set original
	err := km.SetSecret(peerID, originalSecret)
	if err != nil {
		t.Fatalf("SetSecret (original) failed: %v", err)
	}

	// Update with new value
	err = km.SetSecret(peerID, updatedSecret)
	if err != nil {
		t.Fatalf("SetSecret (update) failed: %v", err)
	}

	// Verify updated value
	retrieved, err := km.GetSecret(peerID)
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}

	if retrieved != updatedSecret {
		t.Errorf("Expected updated secret '%s', got '%s'", updatedSecret, retrieved)
	}
}
