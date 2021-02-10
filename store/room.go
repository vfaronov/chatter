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

type Room struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Title   string
	Author  string
	Updated time.Time
	Serial  uint64
}

// fixup updates fields of room in case post was created after room had already
// been fetched from the database.
func (room *Room) fixup(post *Post) {
	if post.Serial > room.Serial {
		room.Serial = post.Serial
	}
	if post.Time.After(room.Updated) {
		room.Updated = post.Time
	}
}

func (db *DB) CreateRoom(ctx context.Context, room *Room) error {
	room.ID = primitive.NilObjectID
	room.Updated = time.Now()
	room.Serial = 1
	res, err := db.rooms.InsertOne(ctx, room)
	if err != nil {
		// TODO: proper error wrapping? (here and elsewhere)
		return err
	}
	room.ID = res.InsertedID.(primitive.ObjectID)
	firstPost := &Post{
		RoomID: room.ID,
		Serial: room.Serial,
		Type:   TypeNewRoom,
		Author: room.Author,
		Time:   room.Updated,
	}
	// TODO: eventually restore consistency if insertPost fails
	// (detect zero posts in GetPosts*)
	return db.insertPost(ctx, firstPost)
}

func (db *DB) GetRoom(ctx context.Context, id primitive.ObjectID) (*Room, error) {
	room := &Room{}
	err := db.rooms.FindOne(ctx, bson.M{"_id": id}).Decode(room)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return room, err
}

func (db *DB) GetRooms(ctx context.Context) ([]*Room, error) {
	cur, err := db.rooms.Find(ctx, bson.M{},
		options.Find().SetSort(bson.M{"updated": -1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rooms []*Room
	for cur.Next(ctx) {
		room := &Room{}
		err = cur.Decode(room)
		if err != nil {
			return rooms, err
		}
		rooms = append(rooms, room)
	}
	return rooms, cur.Err()
}
