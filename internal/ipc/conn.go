package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Conn wraps a net.Conn with JSON line encoding/decoding.
type Conn struct {
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder
}

// NewConn creates a Conn wrapping the given network connection.
func NewConn(c net.Conn) *Conn {
	return &Conn{
		conn: c,
		enc:  json.NewEncoder(c),
		dec:  json.NewDecoder(c),
	}
}

// Send encodes v as JSON and writes it followed by a newline.
func (c *Conn) Send(v any) error {
	return c.enc.Encode(v)
}

// Receive decodes one JSON value from the connection into v.
func (c *Conn) Receive(v any) error {
	return c.dec.Decode(v)
}

// Close closes the underlying connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// Dial connects to a Unix domain socket and returns a Conn.
func Dial(socketPath string, timeout time.Duration) (*Conn, error) {
	conn, err := net.DialTimeout("unix", socketPath, timeout)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", socketPath, err)
	}

	return NewConn(conn), nil
}
