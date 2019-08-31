package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Post struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	RoomID primitive.ObjectID `bson:"room_id"`
	Serial uint32
	Author string
	Time   time.Time
	Text   string
}

func (db *DB) CreatePost(ctx context.Context, post *Post) error {
	post.ID = primitive.NilObjectID
	post.Time = time.Now()

	// Update the room to ensure that it exists, bump its update timestamp,
	// and acquire the serial number for this post. Two posts will never get
	// the same serial number because $inc on one master is atomic.
	res1 := db.rooms.FindOneAndUpdate(ctx,
		bson.M{"_id": post.RoomID},
		bson.M{
			"$set": bson.M{"updated": post.Time},
			"$inc": bson.M{"serial": 1},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	var room struct {
		Serial uint32
	}
	err := res1.Decode(&room)
	switch {
	case err == mongo.ErrNoDocuments:
		return NotFound
	case err != nil:
		return err
	}
	post.Serial = room.Serial

	// Now insert the actual post. If this fails, we end up with slightly
	// inconsistent room data, which is tolerable.
	res, err := db.posts.InsertOne(ctx, post)
	if err != nil {
		return err
	}
	post.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (db *DB) GetPosts(
	ctx context.Context, roomID primitive.ObjectID,
	since, before uint32,
) ([]*Post, error) {
	filter := bson.M{"room_id": roomID}
	if since > 0 {
		filter["serial"] = bson.M{"$gt": since}
	}
	if before > 0 {
		filter["before"] = bson.M{"$lt": before}
	}
	cur, err := db.posts.Find(ctx,
		filter,
		options.Find().SetSort(bson.M{"serial": 1}),
	)
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
