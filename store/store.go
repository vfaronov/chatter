package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectDB(ctx context.Context, uri string) (*DB, error) {
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

	// TODO: timeouts, etc.
	log.Printf("store: connecting to %v", uri)
	db := &DB{}
	db.client, err = mongo.Connect(ctx, options.Client().
		ApplyURI(uri).
		SetAppName("chatter"))
	if err != nil {
		return nil, err
	}
	err = db.client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}
	db.users = db.client.Database(dbname).Collection("users")
	db.rooms = db.client.Database(dbname).Collection("rooms")
	db.posts = db.client.Database(dbname).Collection("posts")

	// TODO: do not run pump in chattertool
	db.listeners.byRoom = make(map[primitive.ObjectID]map[chan *Post]struct{})
	db.listeners.byChannel = make(map[chan *Post]primitive.ObjectID)
	db.listeners.requests = make(chan listenReq, 1024)
	go db.runPump(ctx)

	return db, nil
}

type DB struct {
	client *mongo.Client
	users  *mongo.Collection
	rooms  *mongo.Collection
	posts  *mongo.Collection

	listeners struct {
		// byRoom is for sending a new post to all listening to the room.
		// byChannel is for locating the room ID to detach a listener.
		// These maps are accessed only by the goroutine that consumes requests.
		byRoom    map[primitive.ObjectID]map[chan *Post]struct{}
		byChannel map[chan *Post]primitive.ObjectID
		requests  chan listenReq
	}
}

var (
	ErrNotFound       = errors.New("not found")
	ErrDuplicate      = errors.New("duplicate")
	ErrBadCredentials = errors.New("bad credentials")
)

// isDuplicateKey returns true if err indicates a duplicate key error from MongoDB.
func isDuplicateKey(err error) bool {
	if err, ok := err.(mongo.WriteException); ok {
		for _, err := range err.WriteErrors {
			if err.Code == 11000 {
				return true
			}
		}
	}
	return false
}
