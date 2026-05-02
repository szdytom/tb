package daemon

import (
	"errors"
	"io"
	"log"
	"net"
	"os"

	"github.com/szdytom/tb/internal/ipc"
)

// Serve binds the UDS, sets permissions, and starts the accept loop.
// It is called by Run() and may also be called directly for testing.
func (d *Daemon) Serve() error {
	// Remove stale socket file from previous run.
	_ = os.Remove(d.cfg.SocketPath)

	ln, err := net.Listen("unix", d.cfg.SocketPath)
	if err != nil {
		return err
	}

	if err := os.Chmod(d.cfg.SocketPath, 0600); err != nil {
		_ = ln.Close()

		return err
	}

	d.ln = ln
	log.Printf("IPC server listening on %s", d.cfg.SocketPath)

	go d.acceptLoop(ln)

	return nil
}

// acceptLoop runs in a goroutine, accepting connections and spawning
// a handler goroutine for each one.
func (d *Daemon) acceptLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			log.Printf("accept: %v", err)

			continue
		}

		d.wg.Add(1)

		go d.handleConn(conn)
	}
}

// handleConn reads JSON requests from a single connection, dispatches
// each one, and writes back the response.
func (d *Daemon) handleConn(conn net.Conn) {
	defer d.wg.Done()
	defer conn.Close()

	c := ipc.NewConn(conn)

	for {
		var req ipc.Request
		if err := c.Receive(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}

			log.Printf("read request: %v", err)

			return
		}

		resp := d.dispatch(&req)
		if err := c.Send(resp); err != nil {
			log.Printf("write response: %v", err)

			return
		}
	}
}
