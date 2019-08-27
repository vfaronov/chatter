package web

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/chatter/store"
)

func NewServer(addr string, db *store.DB) *Server {
	r := httprouter.New()
	s := &Server{
		Server: &http.Server{
			Addr:    addr,
			Handler: r,
		},
		db: db,
	}
	r.GET("/rooms/", s.getRooms)
	r.POST("/rooms/", s.postRooms)
	r.GET("/rooms/:roomID", s.withRoom(s.getRoom))
	r.POST("/rooms/:roomID", s.withRoom(s.postRoom))
	return s
}

type Server struct {
	*http.Server
	db *store.DB
}
