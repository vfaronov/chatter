package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"time"

	"github.com/vfaronov/chatter/config"
	"github.com/vfaronov/chatter/store"
	"github.com/vfaronov/chatter/web"
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

	db, err := store.ConnectDB(context.Background(), config.StoreURI)
	if err != nil {
		log.Fatalf("failed to connect to storage DB: %v", err)
	}
	svr := web.NewServer(webAddr, db, []byte(key))

	log.Printf("starting server on %v", webAddr)
	err = svr.ListenAndServe()
	log.Fatalf("server quit: %v", err)
}
