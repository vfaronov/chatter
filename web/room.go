package web

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/chatter/store"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *Server) getRoom(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()
	id, err := primitive.ObjectIDFromHex(ps.ByName("id"))
	if err != nil {
		http.Error(w, "no such room", http.StatusNotFound)
		return
	}
	if _, err := s.db.GetRoom(ctx, id); err == store.NotFound {
		http.Error(w, "no such room", http.StatusNotFound)
		return
	}

	posts, err := s.db.GetPosts(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	for _, post := range posts {
		fmt.Fprint(w, post.Author, ": ", post.Text, "\r\n")
	}
}

func (s *Server) postRoom(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()
	id, err := primitive.ObjectIDFromHex(ps.ByName("id"))
	if err != nil {
		http.Error(w, "no such room", http.StatusNotFound)
		return
	}
	author, _, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("Www-Authenticate", `Basic realm="chatter"`)
		http.Error(w, "need authentication", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("bad form: %s", err), http.StatusBadRequest)
		return
	}
	text := r.Form.Get("text")
	if text == "" {
		http.Error(w, "text required", http.StatusUnprocessableEntity)
		return
	}

	post := store.Post{RoomID: id, Author: author, Text: text}
	err = s.db.AddPost(ctx, post)
	if err == store.NotFound {
		http.Error(w, "no such room", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Location", r.URL.String())
	s.getRoom(w, r, ps)
}
