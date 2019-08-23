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
	r.POST("/", s.postIndex)
	r.GET("/:id", s.getRoom)
	r.POST("/:id", s.postRoom)
	return s
}

type Server struct {
	*http.Server
	db *store.DB
}
