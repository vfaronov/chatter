package web

import (
	"fmt"
	"log"
	"net/http"

	"github.com/vfaronov/chatter/store"
)

func (s *Server) getRoomUpdates(w http.ResponseWriter, r *http.Request, room *store.Room) {
	f, ok := w.(http.Flusher)
	if !ok {
		log.Printf("cannot stream events to %T", w)
		http.Error(w, "cannot stream events", http.StatusNotImplemented)
		return
	}
	posts, err := s.db.StreamPosts(r.Context(), room.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	f.Flush()
	dataw := dataWriter{w}
	for post := range posts {
		_, err = fmt.Fprintf(w, "id: post:%v\ndata: ", post.ID.Hex())
		if err != nil {
			break
		}
		err = roomTpl.ExecuteTemplate(dataw, "post", post)
		if err != nil {
			break
		}
		_, err = w.Write([]byte{'\n', '\n'})
		if err != nil {
			break
		}
		f.Flush()
	}
	if err != nil {
		log.Printf("stopped event stream due to error: %v", err)
	}
}
