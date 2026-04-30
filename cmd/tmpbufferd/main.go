package main

import (
	"log"

	"github.com/szdytom/tb/internal/config"
	"github.com/szdytom/tb/internal/daemon"
)

func main() {
	cfg := config.Default()
	if err := cfg.EnsureDirs(); err != nil {
		log.Fatalf("create directories: %v", err)
	}

	d, err := daemon.New(cfg)
	if err != nil {
		log.Fatalf("start daemon: %v", err)
	}

	if err := d.Run(); err != nil {
		log.Fatalf("daemon error: %v", err)
	}
}
