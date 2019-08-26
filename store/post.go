package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Post struct {
	RoomID primitive.ObjectID
	Author string
	Time   time.Time
	Text   string
}

func (db *DB) AddPost(ctx context.Context, post Post) error {
	post.Time = time.Now()
	res, err := db.rooms.UpdateOne(ctx, bson.M{"_id": post.RoomID},
		bson.M{"$set": bson.M{"updated": post.Time}})
	if err != nil {
		return err
	}
	if res.ModifiedCount < 1 {
		return NotFound
	}
	_, err = db.posts.InsertOne(ctx, post)
	return err
}

func (db *DB) GetPosts(ctx context.Context, roomID primitive.ObjectID) ([]Post, error) {
	cur, err := db.posts.Find(ctx, bson.M{"roomid": roomID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var posts []Post
	for cur.Next(ctx) {
		var post Post
		err = cur.Decode(&post)
		if err != nil {
			return posts, err
		}
		posts = append(posts, post)
	}
	return posts, cur.Err()
}
