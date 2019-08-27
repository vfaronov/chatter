package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Post struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	RoomID primitive.ObjectID `bson:"room_id"`
	Author string
	Time   time.Time
	Text   string
}

func (db *DB) CreatePost(ctx context.Context, post *Post) error {
	post.ID = primitive.NilObjectID
	post.Time = time.Now()

	res1, err := db.rooms.UpdateOne(ctx, bson.M{"_id": post.RoomID},
		bson.M{"$set": bson.M{"updated": post.Time}})
	if err != nil {
		return err
	}
	if res1.ModifiedCount < 1 {
		return NotFound
	}

	res, err := db.posts.InsertOne(ctx, post)
	if err != nil {
		return err
	}
	post.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (db *DB) GetPosts(ctx context.Context, roomID primitive.ObjectID) ([]*Post, error) {
	cur, err := db.posts.Find(ctx, bson.M{"room_id": roomID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var posts []*Post
	for cur.Next(ctx) {
		post := &Post{}
		err = cur.Decode(post)
		if err != nil {
			return posts, err
		}
		posts = append(posts, post)
	}
	return posts, cur.Err()
}
