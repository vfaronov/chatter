package web

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/chatter/store"
)

func (s *Server) getIndex(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	rooms, err := s.db.GetRooms(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	for _, room := range rooms {
		fmt.Fprintf(w, "%s\t[updated %s]\n", room.Title, room.Updated)
	}
}

func (s *Server) postIndex(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("bad form: %s", err), http.StatusBadRequest)
		return
	}
	var room store.Room
	room.Title = r.Form.Get("title")
	if room.Title == "" {
		http.Error(w, "title required", http.StatusUnprocessableEntity)
		return
	}
	id, err := s.db.CreateRoom(r.Context(), room)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	loc := &url.URL{Path: id.Hex()}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Location", loc.String())
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, "See: ", r.URL.ResolveReference(loc), "\r\n")
}
