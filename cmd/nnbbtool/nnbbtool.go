// nnbbtool performs administrative operations on an nnBB instance.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/vfaronov/nnbb/config"
	"github.com/vfaronov/nnbb/store"
)

func main() {
	config.WithStoreURI()
	config.WithFakeData()
	var initDB bool
	flag.BoolVar(&initDB, "init-db", false,
		"initialize collections and indices in the database")
	var insertFake int
	flag.IntVar(&insertFake, "insert-fake", 0,
		"insert fake data into the database with amount `FACTOR` "+
			"(100 is good for development)")
	flag.Parse()

	ctx := context.Background()

	db, err := store.ConnectDB(ctx, config.StoreURI, false)
	if err != nil {
		log.Fatalf("failed to connect to storage DB: %v", err)
	}

	if initDB {
		if err := store.InitDB(ctx, db); err != nil {
			log.Fatalf("failed to init DB: %v", err)
		}
	}
	if insertFake > 0 {
		faker, err := store.NewFaker(config.FakeData)
		if err != nil {
			log.Fatalf("failed to load fake data: %v", err)
		}
		if err := faker.Insert(ctx, db, insertFake); err != nil {
			log.Fatalf("failed to insert fake data: %v", err)
		}
	}
}
