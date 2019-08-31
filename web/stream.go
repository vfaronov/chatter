package web

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

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

	var err error

	var since uint32
	if last := r.Header.Get("Last-Event-Id"); last != "" {
		_, err = fmt.Sscanf(last, "serial:%d", &since)
	} else if err = r.ParseForm(); err == nil {
		var since64 uint64
		since64, err = strconv.ParseUint(r.Form.Get("since"), 10, 64)
		since = uint32(since64)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Subscribe to new posts and let the channel buffer hold them for us
	// while we're catching up with everything already posted since.
	newPosts := s.db.StreamPosts(room.ID)
	defer s.db.StopStreaming(newPosts)

	var posts []*store.Post
	if since > 0 {
		posts, err = s.db.GetPosts(ctx, room.ID, since, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Send response header to the client so it knows we're OK
	// even if we don't have any events to send (yet).
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	f.Flush()

	// Send any posts since.
	var cutoff uint32
	for _, post := range posts {
		err = sendPost(w, post)
		if err != nil {
			log.Printf("stopping stream due to error: %v", err)
			return
		}
		cutoff = post.Serial
	}
	f.Flush()

loop: // Send new posts as they arrive.
	for {
		var post *store.Post
		select {
		case <-ctx.Done(): // client closed connection
			err = ctx.Err()
			break loop
		case post = <-newPosts:
		}
		if post == nil {
			// If we were too slow and our buffer filled up,
			// the pump may have detached us.
			err = errors.New("DB abandoned listener")
			break loop
		}
		if post.Serial <= cutoff {
			// We already got this post from GetPosts.
			continue loop
		}
		err = sendPost(w, post)
		if err != nil {
			break loop
		}
		f.Flush()
	}
	if err != nil {
		log.Printf("stopped event stream due to error: %v", err)
	}
}

// sendPost writes post to w as an HTML fragment in a text/event-stream message.
func sendPost(w http.ResponseWriter, post *store.Post) error {
	_, err := fmt.Fprintf(w, "id: serial:%d\ndata: ", post.Serial)
	if err != nil {
		return err
	}
	err = roomTpl.ExecuteTemplate(dataWriter{w}, "post", post)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte{'\n', '\n'})
	return err
}
