package store

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (db *DB) StreamPosts(ctx context.Context, roomID primitive.ObjectID) (<-chan *Post, error) {
	posts := make(chan *Post)
	go db.streamPosts(ctx, roomID, posts)
	return posts, nil
}

func (db *DB) streamPosts(ctx context.Context, roomID primitive.ObjectID, ch chan<- *Post) {
	defer close(ch)
	log.Printf("start streaming from %v", roomID.Hex())
	cs, err := db.posts.Watch(ctx,
		[]bson.M{{"$match": bson.M{"fullDocument.room_id": roomID}}})
	if err != nil {
		log.Printf("failed to start streaming from %v: %v", roomID.Hex(), err)
		return
	}
	defer cs.Close(ctx)
	// When client closes connection, ctx is canceled and cs.Next returns false.
	for cs.Next(ctx) {
		var data struct {
			Post *Post `bson:"fullDocument"`
		}
		err := cs.Decode(&data)
		if err != nil {
			log.Printf("failed to decode data from %v: %v", roomID.Hex(), err)
			return
		}
		ch <- data.Post
	}
	log.Printf("stop streaming from %v", roomID.Hex())
}
