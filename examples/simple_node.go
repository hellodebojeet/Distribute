package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hellodebojeet/Distribute/internal/dht"
	"github.com/hellodebojeet/Distribute/internal/observability"
	"github.com/hellodebojeet/Distribute/internal/p2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/network"
)

func main() {
	// Create logger
	logger, err := observability.NewLogger(observability.LoggerConfig{
		Level:  "info",
		Format: "json",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	// Create P2P host
	logger.Info("creating P2P host")
	host, privKey, err := p2p.NewHostWithKey("/ip4/0.0.0.0/tcp/4001")
	if err != nil {
		logger.Fatal("failed to create host", observability.ErrorField(err))
	}
	defer host.Close()

	logger.Info("host created",
		observability.StringField("id", host.ID().String()),
		observability.StringField("addrs", fmt.Sprintf("%v", host.Addrs())),
	)

	// Create DHT
	logger.Info("creating DHT")
	dhtConfig := dht.DHTConfig{
		Host: host,
		Mode: kaddht.ModeAutoServer,
	}
	d, err := dht.NewDHT(dhtConfig)
	if err != nil {
		logger.Fatal("failed to create DHT", observability.ErrorField(err))
	}
	defer d.Close()

	// Bootstrap DHT
	logger.Info("bootstrapping DHT")
	ctx := context.Background()
	if err := d.Bootstrap(ctx); err != nil {
		logger.Error("bootstrap failed", observability.ErrorField(err))
	} else {
		logger.Info("DHT bootstrapped successfully")
	}

	// Set up stream handler
	logger.Info("setting up stream handlers")
	host.SetStreamHandler("/distribute/1.0.0", func(s network.Stream) {
		logger.Info("new stream received",
			observability.StringField("peer", s.Conn().RemotePeer().String()),
		)
		s.Close()
	})

	// Create metrics server
	logger.Info("starting metrics server")
	metrics := observability.NewMetrics(observability.MetricsConfig{
		Namespace: "distribute",
	})
	if err := metrics.StartServer(":9090"); err != nil {
		logger.Error("failed to start metrics server", observability.ErrorField(err))
	} else {
		logger.Info("metrics server started", observability.StringField("addr", ":9090"))
	}

	// Print node information
	logger.Info("node started successfully",
		observability.StringField("id", host.ID().String()),
		observability.StringField("listen", ":4001"),
		observability.StringField("metrics", ":9090"),
	)

	// Store private key for future use
	_ = privKey // Private key can be saved to disk for persistence

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutting down node")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := metrics.StopServer(); err != nil {
		logger.Error("failed to stop metrics server", observability.ErrorField(err))
	}

	logger.Info("node shutdown complete")
}
