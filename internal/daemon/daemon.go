package daemon

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/szdytom/tb/internal/config"
	"github.com/szdytom/tb/internal/store"
)

// Daemon manages the tmpbuffer background process lifecycle.
type Daemon struct {
	repo *store.Repository
	db   *store.DB
	cfg  *config.Config
	ln   net.Listener
	wg   sync.WaitGroup
}

// New creates a Daemon, opening the database and running migrations.
func New(cfg *config.Config) (*Daemon, error) {
	db, err := store.Open(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	return &Daemon{repo: store.NewRepository(db), db: db, cfg: cfg}, nil
}

// Run starts the daemon and blocks until SIGINT or SIGTERM.
func (d *Daemon) Run() error {
	log.Printf("tmpbuffer daemon started (pid=%d db=%s)", os.Getpid(), d.cfg.DBPath)

	if err := d.WritePidFile(); err != nil {
		log.Printf("write pid file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.autoPurgeLoop(ctx)

	if err := d.Serve(); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	log.Printf("received %v, shutting down", sig)
	cancel()

	return d.Shutdown()
}

// Shutdown performs a graceful stop.
func (d *Daemon) Shutdown() error {
	log.Print("shutting down daemon")

	d.RemovePidFile()

	if d.ln != nil {
		_ = d.ln.Close()
	}

	d.wg.Wait()

	if d.cfg.SocketPath != "" {
		_ = os.Remove(d.cfg.SocketPath)
	}

	return d.db.Close()
}

// WritePidFile writes the daemon's PID to the configured PID file path.
func (d *Daemon) WritePidFile() error {
	dir := filepath.Dir(d.cfg.PidFilePath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(d.cfg.PidFilePath(), []byte(strconv.Itoa(os.Getpid())), 0600)
}

// RemovePidFile deletes the PID file. Errors are silently ignored.
func (d *Daemon) RemovePidFile() {
	_ = os.Remove(d.cfg.PidFilePath())
}

// autoPurgeLoop periodically removes trashed buffers whose TTL has expired.
func (d *Daemon) autoPurgeLoop(ctx context.Context) {
	d.purgeExpiredTrash()
	_ = d.db.Checkpoint()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.purgeExpiredTrash()
			_ = d.db.Checkpoint()
		case <-ctx.Done():
			return
		}
	}
}

func (d *Daemon) purgeExpiredTrash() {
	n, err := d.repo.DeleteExpiredTrash()
	if err != nil {
		log.Printf("auto-purge: %v", err)

		return
	}

	if n > 0 {
		log.Printf("auto-purge: removed %d expired trash entries", n)
	}
}
