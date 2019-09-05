package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Name         string
	Password     string `bson:"-"`
	PasswordHash string `bson:"passwordHash"`
}

func (u *User) clearSensitive() {
	// Avoid keeping sensitive data in memory.
	u.Password = ""
	u.PasswordHash = ""
}

func (db *DB) CreateUser(ctx context.Context, user *User) error {
	user.ID = primitive.NilObjectID
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password),
		bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("store: cannot generate password hash: %v", err)
	}
	user.PasswordHash = string(hash)
	res, err := db.users.InsertOne(ctx, user)
	if err != nil {
		if isDuplicateKey(err) {
			return ErrDuplicate
		}
		return err
	}
	user.ID = res.InsertedID.(primitive.ObjectID)
	user.clearSensitive()
	return nil
}

// Authenticate checks the Name and Password of user. If the credentials match,
// Authenticate fills the other fields of user and returns nil.
// If the credentials don't match, Authenticate returns ErrBadCredentials.
func (db *DB) Authenticate(ctx context.Context, user *User) error {
	err := db.users.FindOne(ctx, bson.M{"name": user.Name}).Decode(user)
	if err == mongo.ErrNoDocuments {
		return ErrBadCredentials
	}
	if err != nil {
		return err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash),
		[]byte(user.Password))
	if err != nil {
		return ErrBadCredentials
	}
	user.clearSensitive()
	return nil
}
