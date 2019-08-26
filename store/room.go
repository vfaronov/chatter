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
	Title   string
	Updated time.Time
}

func (db *DB) CreateRoom(ctx context.Context, room Room) (primitive.ObjectID, error) {
	room.Updated = time.Now()
	res, err := db.rooms.InsertOne(ctx, room)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return res.InsertedID.(primitive.ObjectID), nil
}

func (db *DB) GetRoom(ctx context.Context, id primitive.ObjectID) (Room, error) {
	var room Room
	err := db.rooms.FindOne(ctx, bson.M{"_id": id}).Decode(&room)
	if err == mongo.ErrNoDocuments {
		err = NotFound
	}
	return room, err
}

func (db *DB) GetRooms(ctx context.Context) ([]Room, error) {
	cur, err := db.rooms.Find(ctx, bson.M{},
		options.Find().SetSort(bson.M{"updated": -1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rooms []Room
	for cur.Next(ctx) {
		var room Room
		err = cur.Decode(&room)
		if err != nil {
			return rooms, err
		}
		rooms = append(rooms, room)
	}
	return rooms, cur.Err()
}
