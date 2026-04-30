package daemon

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/szdytom/tb/internal/config"
	"github.com/szdytom/tb/internal/store"
)

// Daemon manages the tmpbuffer background process lifecycle.
type Daemon struct {
	db  *store.DB
	cfg *config.Config
}

// New creates a Daemon, opening the database and running migrations.
func New(cfg *config.Config) (*Daemon, error) {
	db, err := store.Open(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	return &Daemon{db: db, cfg: cfg}, nil
}

// Run starts the daemon and blocks until SIGINT or SIGTERM.
func (d *Daemon) Run() error {
	log.Printf("tmpbuffer daemon started (pid=%d db=%s)", os.Getpid(), d.cfg.DBPath)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	log.Printf("received %v, shutting down", sig)

	return d.Shutdown()
}

// Shutdown performs a graceful stop.
func (d *Daemon) Shutdown() error {
	log.Print("shutting down daemon")
	return d.db.Close()
}
