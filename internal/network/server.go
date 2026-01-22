package network

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// ConnectionHandler handles messages from a connected peer
type ConnectionHandler interface {
	HandleMessage(conn *PeerConnection, msg *Message)
	OnConnect(conn *PeerConnection)
	OnDisconnect(conn *PeerConnection)
}

// Server handles incoming TCP connections from peers
type Server struct {
	port        int
	listener    net.Listener
	connections map[string]*PeerConnection
	mu          sync.RWMutex
	handler     ConnectionHandler
	ctx         context.Context
	cancel      context.CancelFunc
}

// PeerConnection represents a connection to a remote peer
type PeerConnection struct {
	PeerID       string
	PeerName     string
	Conn         net.Conn
	SharedSecret string
	Paired       bool
	reader       *bufio.Reader
	writer       *bufio.Writer
	writeMu      sync.Mutex
}

// NewServer creates a new TCP server
func NewServer(port int) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		port:        port,
		connections: make(map[string]*PeerConnection),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// SetHandler sets the connection handler
func (s *Server) SetHandler(handler ConnectionHandler) {
	s.handler = handler
}

// Start starts the TCP server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}
	s.listener = listener
	log.Printf("TCP Server: Listening on port %d", s.port)

	go s.acceptLoop()
	return nil
}

// Stop stops the server
func (s *Server) Stop() {
	s.cancel()
	if s.listener != nil {
		s.listener.Close()
	}
	s.mu.Lock()
	for _, conn := range s.connections {
		conn.Close()
	}
	s.mu.Unlock()
}

// acceptLoop accepts incoming connections
func (s *Server) acceptLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if s.ctx.Err() != nil {
					return
				}
				log.Printf("TCP Server: Accept error: %v", err)
				continue
			}
			go s.handleConnection(conn)
		}
	}
}

// handleConnection handles a new incoming connection
func (s *Server) handleConnection(conn net.Conn) {
	peerConn := &PeerConnection{
		Conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}

	// Wait for Hello message to identify peer
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	msg, err := peerConn.ReadMessage()
	if err != nil {
		log.Printf("TCP Server: Failed to read hello: %v", err)
		conn.Close()
		return
	}
	conn.SetReadDeadline(time.Time{})

	if msg.Type != MsgTypeHello {
		log.Printf("TCP Server: Expected hello, got %s", msg.Type)
		conn.Close()
		return
	}

	var hello HelloPayload
	if err := msg.ParsePayload(&hello); err != nil {
		log.Printf("TCP Server: Failed to parse hello: %v", err)
		conn.Close()
		return
	}

	peerConn.PeerID = hello.DeviceID
	peerConn.PeerName = hello.DeviceName

	// Store connection
	s.mu.Lock()
	if existing, ok := s.connections[peerConn.PeerID]; ok {
		existing.Close()
	}
	s.connections[peerConn.PeerID] = peerConn
	s.mu.Unlock()

	log.Printf("TCP Server: Connected to %s (%s)", hello.DeviceName, hello.DeviceID)

	if s.handler != nil {
		s.handler.OnConnect(peerConn)
	}

	// Start reading messages
	s.readLoop(peerConn)
}

// readLoop reads messages from a peer
func (s *Server) readLoop(peerConn *PeerConnection) {
	defer func() {
		s.mu.Lock()
		delete(s.connections, peerConn.PeerID)
		s.mu.Unlock()
		peerConn.Close()
		if s.handler != nil {
			s.handler.OnDisconnect(peerConn)
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			msg, err := peerConn.ReadMessage()
			if err != nil {
				log.Printf("TCP Server: Read error from %s: %v", peerConn.PeerName, err)
				return
			}

			if s.handler != nil {
				s.handler.HandleMessage(peerConn, msg)
			}
		}
	}
}

// GetConnection returns a connection by peer ID
func (s *Server) GetConnection(peerID string) *PeerConnection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connections[peerID]
}

// AddConnection adds a connection to the server
func (s *Server) AddConnection(peerConn *PeerConnection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connections[peerConn.PeerID] = peerConn
}

// ReadMessage reads a message from the connection
func (pc *PeerConnection) ReadMessage() (*Message, error) {
	line, err := pc.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	// Verify HMAC if we have a shared secret
	if pc.SharedSecret != "" && msg.HMAC != "" {
		if !pc.VerifyHMAC(&msg) {
			return nil, fmt.Errorf("HMAC verification failed")
		}
	}

	return &msg, nil
}

// WriteMessage writes a message to the connection
func (pc *PeerConnection) WriteMessage(msg *Message) error {
	pc.writeMu.Lock()
	defer pc.writeMu.Unlock()

	// Sign message if we have a shared secret
	if pc.SharedSecret != "" {
		msg.HMAC = pc.ComputeHMAC(msg)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	data = append(data, '\n')
	if _, err := pc.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return pc.writer.Flush()
}

// Close closes the connection
func (pc *PeerConnection) Close() error {
	if pc.Conn != nil {
		return pc.Conn.Close()
	}
	return nil
}

// ComputeHMAC computes the HMAC for a message
func (pc *PeerConnection) ComputeHMAC(msg *Message) string {
	data := fmt.Sprintf("%s:%d:%s", msg.Type, msg.Timestamp, string(msg.Payload))
	h := hmac.New(sha256.New, []byte(pc.SharedSecret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyHMAC verifies the HMAC of a message
func (pc *PeerConnection) VerifyHMAC(msg *Message) bool {
	expected := pc.ComputeHMAC(msg)
	return hmac.Equal([]byte(expected), []byte(msg.HMAC))
}

// GenerateSharedSecret generates a random shared secret
func GenerateSharedSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// GeneratePairingCode generates a 6-digit pairing code
func GeneratePairingCode() string {
	bytes := make([]byte, 3)
	rand.Read(bytes)
	code := int(bytes[0])<<16 | int(bytes[1])<<8 | int(bytes[2])
	return fmt.Sprintf("%06d", code%1000000)
}
