package network

import (
	"SyncDev/internal/models"
	"encoding/json"
	"time"
)

// MessageType represents the type of protocol message
type MessageType string

const (
	// Connection messages
	MsgTypeHello         MessageType = "hello"
	MsgTypePairingReq    MessageType = "pairing_request"
	MsgTypePairingResp   MessageType = "pairing_response"
	MsgTypeDisconnect    MessageType = "disconnect"

	// Sync messages
	MsgTypeSyncRequest   MessageType = "sync_request"
	MsgTypeSyncResponse  MessageType = "sync_response"
	MsgTypeIndexExchange MessageType = "index_exchange"
	MsgTypeIndexAck      MessageType = "index_ack"

	// File transfer messages
	MsgTypeFileRequest   MessageType = "file_request"
	MsgTypeFileResponse  MessageType = "file_response"
	MsgTypeFileChunk     MessageType = "file_chunk"
	MsgTypeFileComplete  MessageType = "file_complete"
	MsgTypeDeleteFile    MessageType = "delete_file"
	MsgTypeDeleteAck     MessageType = "delete_ack"

	// Status messages
	MsgTypePing          MessageType = "ping"
	MsgTypePong          MessageType = "pong"
	MsgTypeError         MessageType = "error"

	// Config sync messages
	MsgTypeFolderPairSync MessageType = "folder_pair_sync"
)

const (
	// ChunkSize is the size of each file chunk (1MB)
	ChunkSize = 1024 * 1024
	// ProtocolVersion is the current protocol version
	ProtocolVersion = "1.0"
)

// Message is the base structure for all protocol messages
type Message struct {
	Type      MessageType     `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	HMAC      string          `json:"hmac,omitempty"`
}

// HelloPayload is sent when establishing a connection
type HelloPayload struct {
	DeviceID   string `json:"deviceId"`
	DeviceName string `json:"deviceName"`
	Version    string `json:"version"`
}

// PairingRequestPayload is sent to initiate pairing
type PairingRequestPayload struct {
	DeviceID   string `json:"deviceId"`
	DeviceName string `json:"deviceName"`
	Code       string `json:"code"`
}

// PairingResponsePayload is the response to a pairing request
type PairingResponsePayload struct {
	Accepted     bool   `json:"accepted"`
	SharedSecret string `json:"sharedSecret,omitempty"`
	Error        string `json:"error,omitempty"`
}

// SyncRequestPayload requests a sync for a folder pair
type SyncRequestPayload struct {
	FolderPairID string `json:"folderPairId"`
	LocalPath    string `json:"localPath"`
	RemotePath   string `json:"remotePath"`
}

// SyncResponsePayload acknowledges a sync request
type SyncResponsePayload struct {
	FolderPairID string `json:"folderPairId"`
	Accepted     bool   `json:"accepted"`
	Error        string `json:"error,omitempty"`
}

// IndexExchangePayload contains the file index for a folder
type IndexExchangePayload struct {
	FolderPairID string                      `json:"folderPairId"`
	Index        map[string]*models.FileInfo `json:"index"`
}

// FileRequestPayload requests a file from the remote peer
type FileRequestPayload struct {
	FolderPairID string `json:"folderPairId"`
	FilePath     string `json:"filePath"`
	Offset       int64  `json:"offset"`
}

// FileResponsePayload provides metadata about a requested file
type FileResponsePayload struct {
	FolderPairID string `json:"folderPairId"`
	FilePath     string `json:"filePath"`
	Size         int64  `json:"size"`
	Hash         string `json:"hash"`
	Error        string `json:"error,omitempty"`
}

// FileChunkPayload contains a chunk of file data
type FileChunkPayload struct {
	FolderPairID string `json:"folderPairId"`
	FilePath     string `json:"filePath"`
	Offset       int64  `json:"offset"`
	Data         []byte `json:"data"`
	IsLast       bool   `json:"isLast"`
}

// FileCompletePayload signals that a file transfer is complete
type FileCompletePayload struct {
	FolderPairID string `json:"folderPairId"`
	FilePath     string `json:"filePath"`
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
}

// DeleteFilePayload requests deletion of a file
type DeleteFilePayload struct {
	FolderPairID string `json:"folderPairId"`
	FilePath     string `json:"filePath"`
}

// ErrorPayload contains error information
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// FolderPairSyncPayload shares a folder pair configuration with the peer
type FolderPairSyncPayload struct {
	FolderPairID string `json:"folderPairId"`
	LocalPath    string `json:"localPath"`  // Path on the sender's machine
	RemotePath   string `json:"remotePath"` // Path on the receiver's machine
	Action       string `json:"action"`     // "add" or "remove"
}

// NewMessage creates a new protocol message
func NewMessage(msgType MessageType, payload interface{}) (*Message, error) {
	var payloadBytes json.RawMessage
	if payload != nil {
		var err error
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	return &Message{
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payloadBytes,
	}, nil
}

// ParsePayload parses the payload into the given type
func (m *Message) ParsePayload(v interface{}) error {
	if m.Payload == nil {
		return nil
	}
	return json.Unmarshal(m.Payload, v)
}
