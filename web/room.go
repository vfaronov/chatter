package web

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/chatter/store"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	roomTpl = template.Must(template.New("page.html").Funcs(funcMap).ParseFiles(
		"web/templates/page.html",
		"web/templates/room.html",
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
	var err error
	var before, since uint64
	if err = r.ParseForm(); err == nil {
		if s := r.Form.Get("before"); s != "" {
			before, err = strconv.ParseUint(s, 10, 32)
		}
		if s := r.Form.Get("since"); s != "" {
			since, err = strconv.ParseUint(s, 10, 32)
		}
	}
	if before > 0 && since > 0 {
		err = errors.New("cannot specify both before and since")
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("bad query string: %v", err),
			http.StatusBadRequest)
		return
	}
	if before == 0 && since == 0 {
		before = room.Serial + 1
	}

	var posts []*store.Post
	const pageSize = 20
	ctx := r.Context()
	if before > 0 {
		posts, err = s.db.GetPostsBefore(ctx, room.ID, before, pageSize)
	} else {
		posts, err = s.db.GetPostsSince(ctx, room.ID, since, pageSize)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var data struct {
		Room      *store.Room
		Posts     []*store.Post
		FirstPost *store.Post
		LastPost  *store.Post
		Preceding uint64
		Following uint64
	}
	data.Room = room
	data.Posts = posts
	if len(posts) > 0 {
		data.FirstPost = posts[0]
		data.LastPost = posts[len(posts)-1]
		data.Preceding = data.FirstPost.Serial - 1
		data.Following = room.Serial - data.LastPost.Serial
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if isXHR(r) {
		// When we interactively replace the "older posts" fragment of the page,
		// it shouldn't contain the "newer posts" link, and vice-versa.
		if before > 0 {
			data.Following = 0
		} else {
			data.Preceding = 0
		}
		err = roomTpl.ExecuteTemplate(w, "posts", data)
	} else {
		err = roomTpl.Execute(w, data)
	}
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
