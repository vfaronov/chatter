package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Post struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	RoomID primitive.ObjectID `bson:"roomId"`
	Serial uint64
	Author string
	Time   time.Time
	Text   string
}

func (db *DB) CreatePost(ctx context.Context, post *Post) error {
	// Update the room to ensure that it exists, bump its update timestamp,
	// and acquire the serial number for this post. Two posts will never get
	// the same serial number because $inc on one master is atomic.
	post.Time = time.Now()
	res := db.rooms.FindOneAndUpdate(ctx,
		bson.M{"_id": post.RoomID},
		bson.M{
			"$set": bson.M{"updated": post.Time},
			"$inc": bson.M{"serial": 1},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	var room struct {
		Serial uint64
	}
	err := res.Decode(&room)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return ErrNotFound
	case err != nil:
		return err
	}
	post.Serial = room.Serial

	// Now insert the actual post. If this fails, we end up with slightly
	// inconsistent room data, which is tolerable.
	return db.insertPost(ctx, post)
}

func (db *DB) insertPost(ctx context.Context, post *Post) error {
	post.ID = primitive.NilObjectID
	res, err := db.posts.InsertOne(ctx, post)
	if err != nil {
		return err
	}
	post.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (db *DB) GetPostsSince(
	ctx context.Context,
	room *Room,
	since uint64,
	n int64,
) ([]*Post, error) { // TODO: []Post?
	opts := options.Find().SetSort(bson.M{"serial": 1})
	if n > 0 {
		opts = opts.SetLimit(n)
	}
	cur, err := db.posts.Find(ctx,
		bson.M{
			"roomId": room.ID,
			"serial": bson.M{"$gt": since},
		},
		opts,
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	posts := make([]*Post, 0, n)
	for cur.Next(ctx) {
		post := &Post{}
		err = cur.Decode(post)
		if err != nil {
			return posts, err
		}
		posts = append(posts, post)
		room.fixup(post)
	}
	return posts, cur.Err()
}

func (db *DB) GetPostsBefore(
	ctx context.Context,
	room *Room,
	before uint64,
	n int64,
) ([]*Post, error) { // TODO: []Post?
	cur, err := db.posts.Find(ctx,
		bson.M{
			"roomId": room.ID,
			"serial": bson.M{"$lt": before},
		},
		options.Find().
			SetSort(bson.M{"serial": -1}).
			SetLimit(n),
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	// We query MongoDB with reverse sort order, so that we get "20 posts
	// before this one", but we still want to display them in forward order,
	// so we fill in the slice starting from the end.
	posts := make([]*Post, n)
	i := n - 1
	for cur.Next(ctx) {
		post := &Post{}
		err = cur.Decode(post)
		if err != nil {
			return posts, err
		}
		posts[i] = post
		i--
		room.fixup(post)
	}
	return posts[i+1:], cur.Err()
}
