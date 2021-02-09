package web

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/nnbb/store"
)

var roomsTpl = loadPageTemplate("rooms.html")

func (s *Server) getRooms(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	rooms, err := s.db.GetRooms(r.Context())
	if err != nil {
		reqFatalf(w, r, err, "failed to get rooms")
		return
	}
	s.renderPage(w, r, roomsTpl, "Rooms", rooms)
}

func (s *Server) postRooms(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	name, ok := s.userName(r)
	if !ok {
		http.Error(w, "not logged in", http.StatusForbidden)
		return
	}
	room := &store.Room{
		Author: name,
		Title:  r.Form.Get("title"),
	}
	if room.Title == "" {
		http.Error(w, "missing title in form", http.StatusUnprocessableEntity)
		return
	}
	if err := s.db.CreateRoom(r.Context(), room); err != nil {
		reqFatalf(w, r, err, "failed to create room")
		return
	}
	http.Redirect(w, r, room.ID.Hex()+"/", http.StatusSeeOther)
}
