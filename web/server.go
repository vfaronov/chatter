package web

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/chatter/store"
)

func NewServer(addr string, db *store.DB) *Server {
	s := &Server{
		Server: &http.Server{Addr: addr},
		db:     db,
	}

	r := httprouter.New()
	r.GET("/rooms/", s.getRooms)
	r.POST("/rooms/", s.postRooms)
	r.GET("/rooms/:roomID", s.withRoom(s.getRoom))
	r.POST("/rooms/:roomID", s.needAuth(s.withRoom(s.postRoom)))

	s.Server.Handler = s.withAuth(r)

	return s
}

type Server struct {
	*http.Server
	db *store.DB
}

// Private context keys.
type key int

const (
	userKey key = iota
)
