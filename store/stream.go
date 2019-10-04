package store

import (
	"context"
	"errors"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// StreamRoom returns a channel that will receive all new posts to roomID.
// This channel will be closed when its buffer fills up (so it must be read
// in a timely manner) or after a call to CancelStream or CancelStreams.
// Callers must not close the channel themselves.
//
// StreamRoom panics if the stream parameter passed to ConnectDB was false.
func (db *DB) StreamRoom(roomID primitive.ObjectID) chan *Post {
	if db.pump == nil {
		panic("store: StreamRoom called on DB without pump")
	}
	ch := make(chan *Post, 128)
	db.pump.listeners <- listener{true, ch, roomID}
	return ch
}

// CancelStream requests db to stop streaming new posts to ch, and close it.
func (db *DB) CancelStream(ch chan *Post) {
	db.pump.listeners <- listener{attach: false, ch: ch}
}

// CancelStreams requests db to close all channels returned by StreamRoom,
// including any newly returned ones. This method may be called before Disconnect
// in order to gracefully interrupt long-lived user connections without breaking
// the short-lived user requests that are currently in flight.
//
// CancelStreams panics if the stream parameter passed to ConnectDB was false.
func (db *DB) CancelStreams() {
	if db.pump == nil {
		panic("store: CancelStreams called on DB without pump")
	}
	select {
	case db.pump.cancel <- struct{}{}:
		// OK
	default:
		// This may happen if CancelStreams is called multiple times.
		// Because this is an idempotent operation, we only need to send
		// the cancel signal once.
	}
}

// pump dispatches new posts to listeners (SSE handlers).
// DB communicates with pump only by sending on the pump's channels.
type pump struct {
	db        *DB
	posts     chan *Post
	listeners chan listener
	cancel    chan struct{}

	// byRoom is for sending a new post to everyone listening to the room.
	byRoom map[primitive.ObjectID]map[chan *Post]struct{}
	// byChannel is for locating the room ID to detach a listener.
	byChannel map[chan *Post]primitive.ObjectID
}

type listener struct {
	attach bool // false means detach an existing listener
	ch     chan *Post
	roomID primitive.ObjectID
}

func newPump(ctx context.Context, db *DB) (*pump, error) {
	log.Print("store: initializing pump")
	pump := &pump{
		db:        db,
		posts:     make(chan *Post),
		listeners: make(chan listener),
		cancel:    make(chan struct{}, 1),
		byRoom:    make(map[primitive.ObjectID]map[chan *Post]struct{}),
		byChannel: make(map[chan *Post]primitive.ObjectID),
	}
	if err := pump.startStream(ctx); err != nil {
		return nil, err
	}
	go pump.run()
	return pump, nil
}

func (pump *pump) startStream(ctx context.Context) error {
	log.Print("store: starting change stream")
	cs, err := pump.db.posts.Watch(ctx,
		[]bson.M{{"$match": bson.M{"operationType": "insert"}}},
	)
	if err != nil {
		return err
	}
	go pump.runStream(ctx, cs)
	return nil
}

func (pump *pump) runStream(ctx context.Context, cs *mongo.ChangeStream) {
	for cs.Next(ctx) {
		var data struct {
			Post *Post `bson:"fullDocument"`
		}
		err := cs.Decode(&data)
		if err != nil {
			log.Printf("store: failed to decode data from change stream: %v", err)
			continue
		}
		pump.posts <- data.Post
	}
	log.Printf("store: change stream ended: %v", cs.Err())
	cs.Close(ctx)
	close(pump.posts)
}

func (pump *pump) run() {
	var err error
loop:
	for {
		select {
		case post := <-pump.posts:
			if post == nil {
				err = errors.New("change stream ended")
				break loop
			}
			for ch := range pump.byRoom[post.RoomID] {
				pump.trySend(ch, post)
			}

		case l := <-pump.listeners:
			if l.attach {
				pump.attachListener(l.ch, l.roomID)
			} else {
				pump.detachListener(l.ch)
			}

		case <-pump.cancel:
			err = errors.New("asked to cancel")
			break loop
		}
	}

	log.Printf("store: pump winding down: %v", err)
	for ch := range pump.byChannel {
		pump.detachListener(ch)
	}
	// SSE-handling goroutines may keep creating listeners,
	// so we keep draining them and shutting them down.
	// TODO: This never returns, which means this goroutine leaks.
	// This isn't a problem in practice (because we don't ConnectDB
	// more than once per process), but is ugly nonetheless.
	for l := range pump.listeners {
		if l.attach { // have to go through this to avoid double-close
			pump.attachListener(l.ch, l.roomID)
		}
		pump.detachListener(l.ch)
	}
}

func (pump *pump) attachListener(ch chan *Post, roomID primitive.ObjectID) {
	log.Printf("store: attaching listener: %v", ch)
	inRoom := pump.byRoom[roomID]
	if inRoom == nil {
		inRoom = make(map[chan *Post]struct{})
		pump.byRoom[roomID] = inRoom
	}
	inRoom[ch] = struct{}{}
	pump.byChannel[ch] = roomID
}

func (pump *pump) detachListener(ch chan *Post) {
	if roomID, ok := pump.byChannel[ch]; ok {
		log.Printf("store: detaching listener: %v", ch)
		delete(pump.byRoom[roomID], ch)
		delete(pump.byChannel, ch)
		close(ch)
	}
}

// trySend attempts to send post to an attached listener.
// If the listener's buffer is full, trySend detaches it.
func (pump *pump) trySend(ch chan *Post, post *Post) {
	select {
	case ch <- post:
		// OK
	default:
		log.Printf("store: detaching dead listener: %v", ch)
		delete(pump.byRoom[post.RoomID], ch)
		delete(pump.byChannel, ch)
		close(ch)
	}
}
