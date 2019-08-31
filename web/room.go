package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/chatter/store"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	roomTpl = template.Must(template.ParseFiles(
		"web/templates/page.html",
		"web/templates/room.html",
		"web/templates/post.html",
	))
)

func (s *Server) withRoom(
	next func(w http.ResponseWriter, r *http.Request, room *store.Room),
) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id, err := primitive.ObjectIDFromHex(ps.ByName("roomID"))
		if err != nil {
			http.Error(w, "no such room", http.StatusNotFound)
			return
		}
		room, err := s.db.GetRoom(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if room == nil {
			http.Error(w, "no such room", http.StatusNotFound)
			return
		}
		next(w, r, room)
	}
}

func (s *Server) getRoom(w http.ResponseWriter, r *http.Request, room *store.Room) {
	posts, err := s.db.GetPosts(r.Context(), room.ID, 0, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = roomTpl.Execute(w, struct {
		Room     *store.Room
		Posts    []*store.Post
		LastPost *store.Post
	}{
		Room:     room,
		Posts:    posts,
		LastPost: last(posts),
	})
	if err != nil {
		log.Printf("cannot render room: %v", err)
	}
}

func (s *Server) postRoom(w http.ResponseWriter, r *http.Request, room *store.Room) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("bad form: %s", err), http.StatusBadRequest)
		return
	}

	post := &store.Post{
		RoomID: room.ID,
		Author: s.mustUser(r),
		Text:   r.Form.Get("text"),
	}
	if post.Text == "" {
		http.Error(w, "text required", http.StatusUnprocessableEntity)
		return
	}

	if err := s.db.CreatePost(r.Context(), post); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isXHR(r) {
		w.WriteHeader(http.StatusResetContent)
	} else {
		http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
	}
}

func last(posts []*store.Post) *store.Post {
	if len(posts) == 0 {
		return &store.Post{}
	}
	return posts[len(posts)-1]
}
