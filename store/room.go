package store

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Room struct {
	Title string
}

func (db *DB) CreateRoom(ctx context.Context, room Room) (primitive.ObjectID, error) {
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
		return room, NotFound
	}
	if err != nil {
		return room, err
	}
	return room, nil
}
