package web

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/vfaronov/chatter/store"
)

func (s *Server) getRoomUpdates(w http.ResponseWriter, r *http.Request, room *store.Room) {
	ctx := r.Context()
	f, ok := w.(http.Flusher)
	if !ok {
		log.Printf("cannot stream events to %T", w)
		http.Error(w, "cannot stream events", http.StatusNotImplemented)
		return
	}
	posts := s.db.StreamPosts(room.ID)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	f.Flush()
	dataw := dataWriter{w}
	var err error
loop:
	for {
		var post *store.Post
		select {
		case <-ctx.Done(): // client closed connection
			err = ctx.Err()
			break loop
		case post = <-posts:
		}
		if post == nil {
			// If we were too slow and our buffer filled up,
			// the pump may have detached us.
			err = errors.New("DB abandoned listener")
			break loop
		}
		_, err = fmt.Fprintf(w, "id: serial:%v\ndata: ", post.Serial)
		if err != nil {
			break loop
		}
		err = roomTpl.ExecuteTemplate(dataw, "post", post)
		if err != nil {
			break loop
		}
		_, err = w.Write([]byte{'\n', '\n'})
		if err != nil {
			break loop
		}
		f.Flush()
	}
	s.db.StopStreaming(posts)
	if err != nil {
		log.Printf("stopped event stream due to error: %v", err)
	}
}
