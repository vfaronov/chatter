package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Room struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Title   string
	Updated time.Time
	Serial  uint64
}

func (db *DB) CreateRoom(ctx context.Context, room *Room) error {
	room.ID = primitive.NilObjectID
	room.Updated = time.Now()
	res, err := db.rooms.InsertOne(ctx, room)
	if err != nil {
		return err
	}
	room.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (db *DB) GetRoom(ctx context.Context, id primitive.ObjectID) (*Room, error) {
	room := &Room{}
	err := db.rooms.FindOne(ctx, bson.M{"_id": id}).Decode(room)
	if err == mongo.ErrNoDocuments {
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
