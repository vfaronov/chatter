package main

import (
	"context"
	"flag"
	"log"

	"github.com/vfaronov/chatter/config"
	"github.com/vfaronov/chatter/store"
	"github.com/vfaronov/chatter/web"
)

func main() {
	flag.Parse()

	db, err := store.ConnectDB(context.Background(), config.StoreURI)
	if err != nil {
		log.Fatalf("cannot connect to storage DB: %v", err)
	}
	svr := web.NewServer(config.WebAddr, db)

	log.Printf("starting server on %v", config.WebAddr)
	err = svr.ListenAndServe()
	log.Fatalf("shutting down: %v", err)
}
