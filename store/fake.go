package store

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Faker produces fake test data. A zero Faker is ready to generate random texts.
// To use real texts instead, see func NewFaker.
type Faker struct {
	names  []string
	titles []string
	texts  []string
}

// NewFaker returns a Faker that produces data from the given file path.
// If path is empty, returns a zero Faker.
//
// The file at path must contain a stream of JSON objects,
// each object containing one or more of the fields "author", "topic", "text".
// Example file:
//
//	{"topic": "Let's discuss something"}
//	{"author": "Vasiliy", "text": "This is a great idea!"}
//	{"author": "admin"}
//
func NewFaker(path string) (Faker, error) {
	var fk Faker
	if path == "" {
		return fk, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return fk, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	for {
		var entry struct{ Author, Topic, Text string }
		if err := dec.Decode(&entry); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return fk, err
		}
		if entry.Author != "" {
			fk.names = append(fk.names, entry.Author)
		}
		if entry.Topic != "" {
			fk.titles = append(fk.titles, entry.Topic)
		}
		if entry.Text != "" {
			fk.texts = append(fk.texts, entry.Text)
		}
	}
	log.Printf("loaded Faker with %d names, %d titles, %d texts",
		len(fk.names), len(fk.titles), len(fk.texts))
	return fk, nil
}

// Insert generates and inserts fake data into db, spanning some recent
// time range. Exactly factor rooms will be created; the number of posts
// and the length of the time range also increases with factor.
func (fk Faker) Insert(ctx context.Context, db *DB, factor int) error {
	// TODO: fake users
	for i := 0; i < factor; i++ {
		if err := fk.insertFakeRoom(ctx, db, factor); err != nil {
			return err
		}
	}
	return nil
}

func (fk Faker) insertFakeRoom(ctx context.Context, db *DB, factor int) error {
	room := &Room{
		Title:  fk.RoomTitle(),
		Author: fk.UserName(),
	}
	if err := db.CreateRoom(ctx, room); err != nil {
		return err
	}

	// Insert posts directly into the database.
	// Begin up to factor days ago.
	post := &Post{
		RoomID: room.ID,
		Serial: room.Serial,
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
		post.Author = fk.UserName()
		post.Text = fk.PostText()
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

func (fk Faker) UserName() string {
	if len(fk.names) > 0 {
		return oneOf(fk.names...)
	}
	return oneOf(gofakeit.Username(), gofakeit.Name())
}

func (fk Faker) RoomTitle() string {
	if len(fk.titles) > 0 {
		return oneOf(fk.titles...)
	}
	return strings.TrimSuffix(oneOf(
		gofakeit.Word(),
		gofakeit.Sentence(1+rand.Intn(5)),
		gofakeit.HipsterSentence(1+rand.Intn(5)),
	), ".")
}

func (fk Faker) PostText() string {
	if len(fk.texts) > 0 {
		return oneOf(fk.texts...)
	}
	return oneOf(
		gofakeit.Paragraph(1, 1+rand.Intn(5), 1+rand.Intn(10), ""),
		gofakeit.Sentence(1+rand.Intn(5)),
		gofakeit.HipsterParagraph(1, 1+rand.Intn(5), 1+rand.Intn(10), ""),
		gofakeit.HipsterSentence(1+rand.Intn(5)),
		gofakeit.HackerPhrase(),
	)
}

func oneOf(ss ...string) string {
	return ss[rand.Intn(len(ss))]
}
