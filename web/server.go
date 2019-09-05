package web

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/chatter/store"
)

func NewServer(addr string, db *store.DB, key []byte) *Server {
	s := &Server{
		Server: &http.Server{Addr: addr},
		db:     db,
		sess:   sessions.NewCookieStore(key),
	}

	r := httprouter.New()
	r.GET("/chatter.css", s.static("chatter.css"))
	r.GET("/signup/", s.getSignup)
	r.POST("/signup/", s.postSignup)
	r.GET("/rooms/", s.getRooms)
	r.POST("/rooms/", s.postRooms)
	r.GET("/rooms/:roomID/", s.withRoom(s.getRoom))
	r.POST("/rooms/:roomID/", s.withRoom(s.postRoom))
	r.GET("/rooms/:roomID/updates/", s.withRoom(s.getRoomUpdates))

	s.Server.Handler = withReqID(r)

	return s
}

type Server struct {
	*http.Server
	db   *store.DB
	sess *sessions.CookieStore
}

func (s *Server) static(basename string) httprouter.Handle {
	name := "web/static/" + basename
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		http.ServeFile(w, r, name)
	}
}

func isXHR(r *http.Request) bool {
	return r.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

// Private context keys.
type key int

const (
	reqIDKey key = iota
)
