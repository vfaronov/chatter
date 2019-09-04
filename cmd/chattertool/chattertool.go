// Chattertool performs administrative operations on a Chatter instance.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/vfaronov/chatter/config"
	"github.com/vfaronov/chatter/store"
)

func main() {
	var (
		initDB     bool
		insertFake int
	)
	flag.BoolVar(&initDB, "init-db", false,
		"initialize collections and indices in the database")
	flag.IntVar(&insertFake, "insert-fake", 0,
		"insert fake data into the database with the given amount factor "+
			"(100 is good for development)")
	flag.Parse()

	ctx := context.Background()

	db, err := store.ConnectDB(ctx, config.StoreURI)
	if err != nil {
		log.Fatalf("failed to connect to storage DB: %v", err)
	}

	if initDB {
		if err := store.InitDB(ctx, db); err != nil {
			log.Fatalf("failed to init DB: %v", err)
		}
	}
	if insertFake > 0 {
		if err := store.InsertFake(ctx, db, insertFake); err != nil {
			log.Fatalf("failed to insert fake data: %v", err)
		}
	}
}
