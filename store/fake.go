package store

import (
	"context"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InsertFake generates and inserts fake data into db, spanning some recent
// time range. Exactly factor rooms will be created; the number of posts
// and the length of the time range also increases with factor.
func InsertFake(ctx context.Context, db *DB, factor int) error {
	// TODO: fake users
	for i := 0; i < factor; i++ {
		if err := insertFakeRoom(ctx, db, factor); err != nil {
			return err
		}
	}
	return nil
}

func insertFakeRoom(ctx context.Context, db *DB, factor int) error {
	room := &Room{
		Title: strings.TrimSuffix(oneOf(
			gofakeit.Word(),
			gofakeit.Sentence(1+rand.Intn(5)),
			gofakeit.HipsterSentence(1+rand.Intn(5)),
		), "."),
	}
	if err := db.CreateRoom(ctx, room); err != nil {
		return err
	}

	// Insert posts directly into the database.
	// Begin up to factor days ago.
	post := &Post{
		RoomID: room.ID,
	}
	now := time.Now()
	post.Time = now.Add(-time.Duration(1+rand.Intn(factor)) * 24 * time.Hour)
	active := rand.Float64() < 0.1
	for {
		// Move up to 24 hours forward.
		t := post.Time.Add(time.Duration(1+rand.Intn(24*60*60)) * time.Second)
		if t.After(now) {
			break
		}
		post.Time = t
		post.ID = primitive.NilObjectID
		// Some discussions remain perpetually active;
		// others peter out after a number of posts.
		if !active && rand.Float64() < float64(post.Serial)/100 {
			break
		}
		post.Serial++
		post.Author = oneOf(gofakeit.Username(), gofakeit.Name())
		post.Text = oneOf(
			gofakeit.Paragraph(1, 1+rand.Intn(5), 1+rand.Intn(10), ""),
			gofakeit.Sentence(1+rand.Intn(5)),
			gofakeit.HipsterParagraph(1, 1+rand.Intn(5), 1+rand.Intn(10), ""),
			gofakeit.HipsterSentence(1+rand.Intn(5)),
			gofakeit.HackerPhrase(),
		)
		_, err := db.posts.InsertOne(ctx, post)
		if err != nil {
			return err
		}
	}

	// Update room info to reflect the last post.
	room.Updated = post.Time
	room.Serial = post.Serial
	_, err := db.rooms.ReplaceOne(ctx, bson.M{"_id": room.ID}, room)
	if err != nil {
		return err
	}

	log.Printf("store: inserted fake room %v %q with %v posts",
		room.ID.Hex(), room.Title, post.Serial)
	return nil
}

func oneOf(ss ...string) string {
	return ss[rand.Intn(len(ss))]
}
