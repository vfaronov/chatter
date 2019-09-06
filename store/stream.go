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
	db.listeners.requests <- listenReq{true, ch, roomID}
	return ch
}

// StopStreaming stops streaming new posts to ch, and closes it eventually.
func (db *DB) StopStreaming(ch chan *Post) {
	// TODO: can I pass a context.Context to StreamPosts and
	// use that for cancellation instead of explicit call to StopStreaming?
	db.listeners.requests <- listenReq{attach: false, ch: ch}
}

type listenReq struct {
	attach bool // false means detach an existing listener
	ch     chan *Post
	roomID primitive.ObjectID
}

func (db *DB) runPump(ctx context.Context) {
	log.Print("pump starting up")
	cs, err := db.posts.Watch(ctx,
		[]bson.M{{"$match": bson.M{"operationType": "insert"}}},
	)
	if err != nil {
		log.Printf("failed to start change stream: %v", err)
		return
	}
	defer cs.Close(ctx)
	for cs.Next(ctx) {
		db.processListenReqs()
		var data struct {
			Post *Post `bson:"fullDocument"`
		}
		err := cs.Decode(&data)
		if err != nil {
			log.Printf("failed to decode data from change stream: %v", err)
			continue
		}
		for ch := range db.listeners.byRoom[data.Post.RoomID] {
			db.trySend(ch, data.Post)
		}
	}
	log.Printf("pump shutting down: %v", cs.Err())
}

// processListenReqs handles all listen requests that may have queued up.
func (db *DB) processListenReqs() {
	for {
		select {
		case req := <-db.listeners.requests:
			if req.attach {
				log.Printf("attaching listener: %v", req.ch)
				if db.listeners.byRoom[req.roomID] == nil {
					db.listeners.byRoom[req.roomID] = make(map[chan *Post]struct{})
				}
				db.listeners.byRoom[req.roomID][req.ch] = struct{}{}
				db.listeners.byChannel[req.ch] = req.roomID
			} else {
				log.Printf("detaching listener: %v", req.ch)
				if roomID, ok := db.listeners.byChannel[req.ch]; ok {
					delete(db.listeners.byRoom[roomID], req.ch)
					delete(db.listeners.byChannel, req.ch)
					close(req.ch)
				}
			}
		default:
			return
		}
	}
}

// trySend attempts to send post to an attached listener.
// If the listener's buffer is full, trySend detaches it.
func (db *DB) trySend(ch chan *Post, post *Post) {
	select {
	case ch <- post:
		// OK
	default:
		log.Printf("detaching dead listener: %v", ch)
		delete(db.listeners.byRoom[post.RoomID], ch)
		delete(db.listeners.byChannel, ch)
		close(ch)
	}
}
