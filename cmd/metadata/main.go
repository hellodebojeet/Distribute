package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"

	"github.com/hellodebojeet/Distribute/metadata"
)

func main() {
	// Define flags
	listenAddr := flag.String("listen-addr", ":8080", "Address to listen on")
	persistencePath := flag.String("persistence-path", "./metadata.json", "Path to persist metadata")
	persistenceInterval := flag.Duration("persistence-interval", 30*time.Second, "Interval to persist metadata")
	replicationFactor := flag.Int("replication-factor", 3, "Default replication factor")
	chunkSize := flag.Int64("chunk-size", 4*1024*1024, "Target chunk size in bytes (4MB)")
	nodeTimeout := flag.Duration("node-timeout", 2*time.Minute, "Time after which node is considered dead")
	flag.Parse()

	// Create metadata store config
	config := metadata.Config{
		ReplicationFactor:   *replicationFactor,
		ChunkSize:           *chunkSize,
		PersistencePath:     *persistencePath,
		PersistenceInterval: *persistenceInterval,
		NodeTimeout:         *nodeTimeout,
	}

	// Create metadata store
	store, err := metadata.NewInMemoryMetadataStore(config)
	if err != nil {
		log.Fatalf("Failed to create metadata store: %v", err)
	}
	defer store.Close()

	// Create handler
	handler := metadata.NewHandler(store, config)

	// Create router
	r := mux.NewRouter()
	handler.RegisterRoutes(r)

	// Add middleware for logging and CORS if needed
	r.Use(loggingMiddleware)

	// Create HTTP server
	srv := &http.Server{
		Handler: r,
		Addr:    *listenAddr,
		// Good practice: enforce timeouts for servers created
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting metadata service on %s", *listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("Shutting down metadata service")
	os.Exit(0)
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
