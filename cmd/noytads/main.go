package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/caf3babe/noytads/internal/proxy"
)

func main() {
	listenAddr := flag.String("listen", ":8443", "Proxy listen address")
	caPath := flag.String("ca-path", "/etc/noytads/ca.pem", "Path to CA certificate")
	caKeyPath := flag.String("ca-key", "/etc/noytads/ca.key", "Path to CA private key")
	caExportPort := flag.String("ca-export", ":8080", "Port to serve CA cert for iOS install")
	logLevel := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	flag.Parse()

	cfg := &proxy.Config{
		ListenAddr:   *listenAddr,
		CAPath:       *caPath,
		CAKeyPath:    *caKeyPath,
		CAExportPort: *caExportPort,
		LogLevel:     *logLevel,
	}

	srv, err := proxy.NewServer(cfg)
	if err != nil {
		log.Fatalf("failed to create proxy server: %v", err)
	}

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("shutting down...")
		srv.Shutdown()
		os.Exit(0)
	}()

	log.Printf("noytads starting on %s", *listenAddr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("proxy server failed: %v", err)
	}
}
