package web

import (
	"errors"
	"fmt"
	"html/template"
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
		ctx := r.Context()
		id, err := primitive.ObjectIDFromHex(ps.ByName("roomID"))
		if err != nil {
			http.Error(w, "no such room", http.StatusNotFound)
			return
		}
		room, err := s.db.GetRoom(ctx, id)
		if err == store.ErrNotFound {
			http.Error(w, "no such room", http.StatusNotFound)
			return
		}
		if err != nil {
			reqFatalf(w, r, err, "failed to get room")
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
	if s := r.Form.Get("before"); s != "" {
		before, err = strconv.ParseUint(s, 10, 32)
	}
	if s := r.Form.Get("since"); s != "" {
		since, err = strconv.ParseUint(s, 10, 32)
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
		reqFatalf(w, r, err, "failed to get posts")
		return
	}

	fragment := isXHR(r)
	keyval := []interface{}{
		"Room", room,
		"Posts", posts,
	}
	if len(posts) > 0 {
		firstPost := posts[0]
		lastPost := posts[len(posts)-1]
		preceding := firstPost.Serial - 1
		following := room.Serial - lastPost.Serial
		if fragment {
			// When we interactively replace the "older posts" fragment
			// of the page, it shouldn't contain the "newer posts" link,
			// and vice-versa.
			if before > 0 {
				following = 0
			} else {
				preceding = 0
			}
		}
		keyval = append(keyval,
			"FirstPost", firstPost,
			"LastPost", lastPost,
			"Preceding", preceding,
			"Following", following,
		)
	}
	if fragment {
		s.renderFragment(w, r, roomTpl, "posts", keyval...)
	} else {
		s.renderPage(w, r, roomTpl, keyval...)
	}
}

func (s *Server) postRoom(w http.ResponseWriter, r *http.Request, room *store.Room) {
	userName, ok := s.userName(r)
	if !ok {
		http.Error(w, "not logged in", http.StatusForbidden)
		return
	}
	post := &store.Post{
		RoomID: room.ID,
		Author: userName,
		Text:   r.Form.Get("text"),
	}
	if post.Text == "" {
		http.Error(w, "text required", http.StatusUnprocessableEntity)
		return
	}

	if err := s.db.CreatePost(r.Context(), post); err != nil {
		reqFatalf(w, r, err, "failed to create post")
		return
	}

	if isXHR(r) {
		w.WriteHeader(http.StatusResetContent)
	} else {
		http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
	}
}
