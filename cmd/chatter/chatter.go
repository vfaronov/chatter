package main

import (
	"context"
	"flag"
	"log"

	"github.com/vfaronov/chatter/store"
	"github.com/vfaronov/chatter/web"
)

func main() {
	var storeURI string
	flag.StringVar(&storeURI, "store-uri",
		"mongodb://localhost:27017/chatter?replicaSet=chatter", "")
	var webAddr string
	flag.StringVar(&webAddr, "web-addr", "localhost:10242", "")
	flag.Parse()

	db, err := store.ConnectDB(context.Background(), storeURI)
	if err != nil {
		log.Fatalf("cannot connect to storage DB: %v", err)
	}
	svr := web.NewServer(webAddr, db)

	log.Printf("starting server on %v", webAddr)
	err = svr.ListenAndServe()
	log.Fatalf("shutting down: %v", err)
}
