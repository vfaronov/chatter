package store

import (
	"context"
	"errors"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectDB(uri string) (*DB, error) {
	db := &DB{}
	var err error
	log.Printf("connecting to storage DB at %v", uri)
	db.client, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	// TODO: Ping?
	db.rooms = db.client.Database("chatter").Collection("rooms") // FIXME
	db.posts = db.client.Database("chatter").Collection("posts") // FIXME
	return db, nil
}

type DB struct {
	client *mongo.Client
	rooms  *mongo.Collection
	posts  *mongo.Collection
}

var NotFound = errors.New("not found")
