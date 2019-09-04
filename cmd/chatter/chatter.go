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

	flag.Parse()

	db, err := store.ConnectDB(context.Background(), config.StoreURI)
	if err != nil {
		log.Fatalf("failed to connect to storage DB: %v", err)
	}
	svr := web.NewServer(config.WebAddr, db)

	log.Printf("starting server on %v", config.WebAddr)
	err = svr.ListenAndServe()
	log.Fatalf("server quit: %v", err)
}
