package store

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamPosts returns a channel that receives all new posts to roomID.
// This channel must be read in a timely manner; if its buffer fills up,
// it will be closed. Callers must not close the channel themselves.
// To cancel streaming, call StopStreaming with the same channel.
func (db *DB) StreamPosts(roomID primitive.ObjectID) chan *Post {
	ch := make(chan *Post, 128)
	db.attach <- listener{ch, roomID}
	return ch
}

// StopStreaming stops streaming new posts to ch, and closes it eventually.
func (db *DB) StopStreaming(ch chan *Post) {
	db.attach <- listener{ch, primitive.NilObjectID}
}

type listener struct {
	ch     chan *Post
	roomID primitive.ObjectID // zero is used to request detach
}

func (db *DB) runPump(ctx context.Context) {
	log.Print("pump starting up")
	cs, err := db.posts.Watch(ctx,
		[]bson.M{{"$match": bson.M{"operationType": "insert"}}},
	)
	if err != nil {
		log.Printf("cannot start change stream: %v", err)
		return
	}
	defer cs.Close(ctx)
	for cs.Next(ctx) {
		db.processListeners()
		var data struct {
			Post *Post `bson:"fullDocument"`
		}
		err := cs.Decode(&data)
		if err != nil {
			log.Printf("cannot decode data from change stream: %v", err)
			continue
		}
		for _, l := range db.listeners {
			if l.roomID == data.Post.RoomID {
				db.trySend(l.ch, data.Post)
			}
		}
	}
	log.Printf("pump shutting down: %v", cs.Err())
}

// processListeners handles all attach/detach requests that may have queued up.
func (db *DB) processListeners() {
	for {
		select {
		case l := <-db.attach:
			if l.roomID.IsZero() {
				log.Printf("detaching listener: %v", l.ch)
				if _, ok := db.listeners[l.ch]; ok {
					delete(db.listeners, l.ch)
					close(l.ch)
				}
			} else {
				log.Printf("attaching listener: %v", l.ch)
				db.listeners[l.ch] = l
			}
		default:
			return
		}
	}
}

// trySend attempts to send a new post to an attached listener.
// If the listener's buffer is full, trySend detaches it.
func (db *DB) trySend(ch chan *Post, post *Post) {
	select {
	case ch <- post:
		// OK
	default:
		log.Printf("detaching dead listener: %v", ch)
		delete(db.listeners, ch)
		close(ch)
	}
}
