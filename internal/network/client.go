package network

import (
	"SyncDev/internal/config"
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

// Client handles outgoing TCP connections to peers
type Client struct {
	deviceID   string
	deviceName string
}

// NewClient creates a new TCP client
func NewClient(deviceID, deviceName string) *Client {
	return &Client{
		deviceID:   deviceID,
		deviceName: deviceName,
	}
}

// Connect establishes a connection to a peer
func (c *Client) Connect(host string, port int) (*PeerConnection, error) {
	return c.ConnectWithContext(context.Background(), host, port)
}

// ConnectWithContext establishes a connection with a context
func (c *Client) ConnectWithContext(ctx context.Context, host string, port int) (*PeerConnection, error) {
	addr := fmt.Sprintf("%s:%d", host, port)

	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	peerConn := &PeerConnection{
		Conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}

	// Send Hello message
	hello := &HelloPayload{
		DeviceID:   c.deviceID,
		DeviceName: c.deviceName,
		Version:    config.AppVersion,
	}

	msg, err := NewMessage(MsgTypeHello, hello)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create hello message: %w", err)
	}

	if err := peerConn.WriteMessage(msg); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send hello: %w", err)
	}

	log.Printf("TCP Client: Connected to %s", addr)
	return peerConn, nil
}

// SendPairingRequest sends a pairing request to a peer
func (c *Client) SendPairingRequest(peerConn *PeerConnection, code string) error {
	payload := &PairingRequestPayload{
		DeviceID:   c.deviceID,
		DeviceName: c.deviceName,
		Code:       code,
	}

	msg, err := NewMessage(MsgTypePairingReq, payload)
	if err != nil {
		return err
	}

	return peerConn.WriteMessage(msg)
}

// SendPairingResponse sends a pairing response
func (c *Client) SendPairingResponse(peerConn *PeerConnection, accepted bool, sharedSecret string, errMsg string) error {
	payload := &PairingResponsePayload{
		Accepted:     accepted,
		SharedSecret: sharedSecret,
		Error:        errMsg,
	}

	msg, err := NewMessage(MsgTypePairingResp, payload)
	if err != nil {
		return err
	}

	return peerConn.WriteMessage(msg)
}

// SendSyncRequest sends a sync request for a folder pair
func (c *Client) SendSyncRequest(peerConn *PeerConnection, folderPairID, localPath, remotePath string) error {
	payload := &SyncRequestPayload{
		FolderPairID: folderPairID,
		LocalPath:    localPath,
		RemotePath:   remotePath,
	}

	msg, err := NewMessage(MsgTypeSyncRequest, payload)
	if err != nil {
		return err
	}

	return peerConn.WriteMessage(msg)
}

// SendIndexExchange sends a file index to the peer
func (c *Client) SendIndexExchange(peerConn *PeerConnection, payload *IndexExchangePayload) error {
	msg, err := NewMessage(MsgTypeIndexExchange, payload)
	if err != nil {
		return err
	}

	return peerConn.WriteMessage(msg)
}

// SendFileRequest requests a file from the peer
func (c *Client) SendFileRequest(peerConn *PeerConnection, folderPairID, filePath string, offset int64) error {
	payload := &FileRequestPayload{
		FolderPairID: folderPairID,
		FilePath:     filePath,
		Offset:       offset,
	}

	msg, err := NewMessage(MsgTypeFileRequest, payload)
	if err != nil {
		return err
	}

	return peerConn.WriteMessage(msg)
}

// SendFileChunk sends a file chunk to the peer
func (c *Client) SendFileChunk(peerConn *PeerConnection, payload *FileChunkPayload) error {
	msg, err := NewMessage(MsgTypeFileChunk, payload)
	if err != nil {
		return err
	}

	return peerConn.WriteMessage(msg)
}

// SendFileComplete signals file transfer completion
func (c *Client) SendFileComplete(peerConn *PeerConnection, folderPairID, filePath string, success bool, errMsg string) error {
	payload := &FileCompletePayload{
		FolderPairID: folderPairID,
		FilePath:     filePath,
		Success:      success,
		Error:        errMsg,
	}

	msg, err := NewMessage(MsgTypeFileComplete, payload)
	if err != nil {
		return err
	}

	return peerConn.WriteMessage(msg)
}

// SendDeleteFile requests deletion of a file
func (c *Client) SendDeleteFile(peerConn *PeerConnection, folderPairID, filePath string) error {
	payload := &DeleteFilePayload{
		FolderPairID: folderPairID,
		FilePath:     filePath,
	}

	msg, err := NewMessage(MsgTypeDeleteFile, payload)
	if err != nil {
		return err
	}

	return peerConn.WriteMessage(msg)
}

// SendPing sends a ping message
func (c *Client) SendPing(peerConn *PeerConnection) error {
	msg, err := NewMessage(MsgTypePing, nil)
	if err != nil {
		return err
	}

	return peerConn.WriteMessage(msg)
}

// SendPong sends a pong response
func (c *Client) SendPong(peerConn *PeerConnection) error {
	msg, err := NewMessage(MsgTypePong, nil)
	if err != nil {
		return err
	}

	return peerConn.WriteMessage(msg)
}

// SendError sends an error message
func (c *Client) SendError(peerConn *PeerConnection, code, message string) error {
	payload := &ErrorPayload{
		Code:    code,
		Message: message,
	}

	msg, err := NewMessage(MsgTypeError, payload)
	if err != nil {
		return err
	}

	return peerConn.WriteMessage(msg)
}
