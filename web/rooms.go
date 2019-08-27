package web

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/chatter/store"
)

var (
	roomsTpl = template.Must(template.ParseFiles(
		"web/templates/page.html",
		"web/templates/rooms.html"))
)

func (s *Server) getRooms(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	rooms, err := s.db.GetRooms(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = roomsTpl.Execute(w, rooms)
}

func (s *Server) postRooms(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("bad form: %s", err), http.StatusBadRequest)
		return
	}
	room := &store.Room{}
	room.Title = r.Form.Get("title")
	if room.Title == "" {
		http.Error(w, "title required", http.StatusUnprocessableEntity)
		return
	}
	if err := s.db.CreateRoom(r.Context(), room); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, room.ID.Hex(), http.StatusSeeOther)
}
