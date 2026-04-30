package cli

import (
	"fmt"

	"github.com/szdytom/tb/internal/buffer"
	"github.com/szdytom/tb/internal/config"
	"github.com/szdytom/tb/internal/daemon"
	"github.com/szdytom/tb/internal/ipc"
	"github.com/szdytom/tb/internal/store"
)

// Client wraps an IPC connection to the daemon with typed convenience methods.
type Client struct {
	conn   *ipc.Conn
	nextID int64
}

// NewClient connects to the daemon, auto-starting it if necessary.
func NewClient(cfg *config.Config) (*Client, error) {
	conn, err := daemon.Autostart(cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to daemon: %w", err)
	}
	return &Client{conn: conn, nextID: 1}, nil
}

// Close shuts down the underlying connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) do(op ipc.Op, payload interface{}) (ipc.Response, error) {
	req := ipc.NewRequest(c.nextID, op, payload)
	c.nextID++
	if err := c.conn.Send(req); err != nil {
		return ipc.Response{}, fmt.Errorf("send: %w", err)
	}
	var resp ipc.Response
	if err := c.conn.Receive(&resp); err != nil {
		return ipc.Response{}, fmt.Errorf("receive: %w", err)
	}
	if !resp.Ok {
		return resp, fmt.Errorf("%s", resp.Error)
	}
	return resp, nil
}

// Ping checks that the daemon is alive.
func (c *Client) Ping() error {
	resp, err := c.do(ipc.OpPing, nil)
	if err != nil {
		return err
	}
	var p ipc.PingResponse
	if err := resp.UnmarshalPayload(&p); err != nil {
		return fmt.Errorf("decode ping response: %w", err)
	}
	return nil
}

// CreateBuffer creates a new buffer and returns its ID.
func (c *Client) CreateBuffer(content, label string, tags []string) (int64, error) {
	resp, err := c.do(ipc.OpCreateBuffer, ipc.CreateBufferPayload{
		Content: content,
		Label:   label,
		Tags:    tags,
	})
	if err != nil {
		return 0, err
	}
	var idResp ipc.IDResponse
	if err := resp.UnmarshalPayload(&idResp); err != nil {
		return 0, fmt.Errorf("decode create response: %w", err)
	}
	return idResp.ID, nil
}

// GetBuffer retrieves a single buffer by ID.
func (c *Client) GetBuffer(id int64) (*buffer.Buffer, error) {
	resp, err := c.do(ipc.OpGetBuffer, ipc.IDPayload{ID: id})
	if err != nil {
		return nil, err
	}
	var buf buffer.Buffer
	if err := resp.UnmarshalPayload(&buf); err != nil {
		return nil, fmt.Errorf("decode get response: %w", err)
	}
	return &buf, nil
}

// ListBuffers returns active buffers matching the given filter.
func (c *Client) ListBuffers(payload ipc.ListBuffersPayload) ([]*buffer.Buffer, error) {
	resp, err := c.do(ipc.OpListBuffers, payload)
	if err != nil {
		return nil, err
	}
	var bufs []*buffer.Buffer
	if err := resp.UnmarshalPayload(&bufs); err != nil {
		return nil, fmt.Errorf("decode list response: %w", err)
	}
	return bufs, nil
}

// UpdateContent replaces a buffer's content.
func (c *Client) UpdateContent(id int64, content string) error {
	_, err := c.do(ipc.OpUpdateContent, ipc.UpdateContentPayload{
		ID:      id,
		Content: content,
	})
	return err
}

// UpdateLabel changes a buffer's label.
func (c *Client) UpdateLabel(id int64, label string) error {
	_, err := c.do(ipc.OpUpdateLabel, ipc.UpdateLabelPayload{
		ID:    id,
		Label: label,
	})
	return err
}

// UpdateTags replaces a buffer's tags.
func (c *Client) UpdateTags(id int64, tags []string) error {
	_, err := c.do(ipc.OpUpdateTags, ipc.UpdateTagsPayload{
		ID:   id,
		Tags: tags,
	})
	return err
}

// SoftDelete moves a buffer to the trash.
func (c *Client) SoftDelete(id int64, ttlSeconds int) error {
	_, err := c.do(ipc.OpSoftDelete, ipc.SoftDeletePayload{
		ID:         id,
		TTLSeconds: ttlSeconds,
	})
	return err
}

// PermanentlyDelete removes a buffer entirely.
func (c *Client) PermanentlyDelete(id int64) error {
	_, err := c.do(ipc.OpPermanentlyDelete, ipc.IDPayload{ID: id})
	return err
}

// ListTrash returns all trashed buffers.
func (c *Client) ListTrash() ([]*buffer.Buffer, error) {
	resp, err := c.do(ipc.OpListTrash, nil)
	if err != nil {
		return nil, err
	}
	var bufs []*buffer.Buffer
	if err := resp.UnmarshalPayload(&bufs); err != nil {
		return nil, fmt.Errorf("decode list trash response: %w", err)
	}
	return bufs, nil
}

// RestoreFromTrash restores a trashed buffer.
func (c *Client) RestoreFromTrash(id int64) error {
	_, err := c.do(ipc.OpRestoreFromTrash, ipc.IDPayload{ID: id})
	return err
}

// Search performs full-text search across all active buffers.
func (c *Client) Search(query string, isRegex bool) ([]store.SearchResult, error) {
	resp, err := c.do(ipc.OpSearch, ipc.SearchPayload{
		Query:   query,
		IsRegex: isRegex,
	})
	if err != nil {
		return nil, err
	}
	var results []store.SearchResult
	if err := resp.UnmarshalPayload(&results); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	return results, nil
}

// Count returns the number of active buffers.
func (c *Client) Count() (int, error) {
	resp, err := c.do(ipc.OpCount, nil)
	if err != nil {
		return 0, err
	}
	var countResp ipc.CountResponse
	if err := resp.UnmarshalPayload(&countResp); err != nil {
		return 0, fmt.Errorf("decode count response: %w", err)
	}
	return countResp.Count, nil
}
