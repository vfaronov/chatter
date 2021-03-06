package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/nnbb/store"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var roomTpl = loadPageTemplate("room.html")

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
		if errors.Is(err, store.ErrNotFound) {
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

type roomPayload struct {
	Room                 *store.Room
	Posts                []*store.Post
	FirstPost, LastPost  *store.Post
	Preceding, Following uint64
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
		posts, err = s.db.GetPostsBefore(ctx, room, before, pageSize)
	} else {
		posts, err = s.db.GetPostsSince(ctx, room, since, pageSize)
	}
	if err != nil {
		reqFatalf(w, r, err, "failed to get posts")
		return
	}

	fragment := isXHR(r)
	payload := roomPayload{
		Room:  room,
		Posts: posts,
	}
	if len(posts) > 0 {
		payload.FirstPost = posts[0]
		payload.LastPost = posts[len(posts)-1]
		payload.Preceding = payload.FirstPost.Serial - 1
		payload.Following = room.Serial - payload.LastPost.Serial
		if fragment {
			// When we interactively replace the "older posts" fragment
			// of the page, it shouldn't contain the "newer posts" link,
			// and vice-versa.
			if before > 0 {
				payload.Following = 0
			} else {
				payload.Preceding = 0
			}
		}
	}
	if fragment {
		s.renderFragment(w, r, roomTpl, "posts", payload)
	} else {
		s.renderPage(w, r, roomTpl, payload)
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
		s.renderFragment(w, r, roomTpl, "postform", roomPayload{})
	} else {
		http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
	}
}
