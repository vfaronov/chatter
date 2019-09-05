package store

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InitDB initializes collections and indexes in db.
// If any of the indexes already exists, InitDB returns an error.
func InitDB(ctx context.Context, db *DB) error {
	var err error

	log.Print("store: creating index for users")
	_, err = db.users.Indexes().CreateOne(ctx,
		mongo.IndexModel{
			Keys:    bson.M{"name": 1},
			Options: options.Index().SetUnique(true),
		},
	)
	if err != nil {
		return err
	}

	log.Print("store: creating index for rooms")
	_, err = db.rooms.Indexes().CreateOne(ctx,
		mongo.IndexModel{
			Keys: bson.M{"updated": 1},
		},
	)
	if err != nil {
		return err
	}

	log.Print("store: creating index for posts")
	_, err = db.posts.Indexes().CreateOne(ctx,
		mongo.IndexModel{
			Keys:    bson.M{"roomId": 1, "serial": 1},
			Options: options.Index().SetUnique(true),
		},
	)
	if err != nil {
		return err
	}

	return nil
}
