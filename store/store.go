package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectDB(uri string) (*DB, error) {
	// MongoDB's connection string URIs include database name:
	// https://docs.mongodb.com/manual/reference/connection-string/ --
	// but the driver only uses it for authentication. To avoid duplicating
	// the database name in another config option, extract it manually.
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("store: bad URI: %q: %v", uri, err)
	}
	dbname := strings.TrimPrefix(parsedURI.Path, "/")
	if dbname == "" {
		return nil, fmt.Errorf("store: missing database name in URI: %q", uri)
	}

	log.Printf("store: connecting to %v", uri)
	db := &DB{}
	db.client, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	// TODO: Ping?
	db.rooms = db.client.Database(dbname).Collection("rooms")
	db.posts = db.client.Database(dbname).Collection("posts")
	return db, nil
}

type DB struct {
	client *mongo.Client
	rooms  *mongo.Collection
	posts  *mongo.Collection
}

var NotFound = errors.New("not found")
