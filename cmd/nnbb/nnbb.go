package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vfaronov/nnbb/config"
	"github.com/vfaronov/nnbb/store"
	"github.com/vfaronov/nnbb/web"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	var (
		webAddr string
		key     string
	)
	flag.StringVar(&webAddr, "web-addr", "localhost:10242",
		"address for the Web server to listen on")
	flag.StringVar(&key, "key", "", "secret key for cookie signing")
	flag.Parse()

	if key == "" {
		log.Fatalf("no key for cookie signing")
	}

	db, err := store.ConnectDB(context.Background(), config.StoreURI, true)
	if err != nil {
		log.Fatalf("failed to connect to storage DB: %v", err)
	}
	svr := web.NewServer(webAddr, db, []byte(key))

	go runServer(svr)
	handleSignals(svr, db)
}

func runServer(svr *web.Server) {
	log.Printf("starting server on %v", svr.Addr)
	if err := svr.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server quit: %v", err)
	}
}

func handleSignals(svr *web.Server, db *store.DB) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	log.Printf("shutting down server due to signal: %v", sig)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Interrupt all SSE connections to allow svr.Shutdown to proceed
	// without breaking any short-lived requests that are still in flight.
	db.CancelStreams()
	if err := svr.Shutdown(ctx); err != nil {
		log.Printf("failed to shut HTTP server down gracefully: %v", err)
	}
	db.Disconnect(ctx)
}
