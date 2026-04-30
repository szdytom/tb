package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.autoPurgeLoop(ctx)

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
	return d.db.Close()
}

// autoPurgeLoop periodically removes trashed buffers whose TTL has expired.
func (d *Daemon) autoPurgeLoop(ctx context.Context) {
	// Run once immediately on start.
	d.purgeExpiredTrash()
	d.db.Checkpoint()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.purgeExpiredTrash()
			d.db.Checkpoint()
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
