package web

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/chatter/store"
)

func NewServer(addr string, db *store.DB, key []byte) *Server {
	s := &Server{
		Server:       &http.Server{Addr: addr},
		db:           db,
		sessionStore: sessions.NewCookieStore(key),
	}

	r := httprouter.New()
	r.GET("/chatter.css", s.static("chatter.css"))
	r.GET("/signup/", s.getSignup)
	r.POST("/signup/", s.postSignup)
	r.POST("/logout/", s.postLogout)
	r.GET("/rooms/", s.getRooms)
	r.POST("/rooms/", s.postRooms)
	r.GET("/rooms/:roomID/", s.withRoom(s.getRoom))
	r.POST("/rooms/:roomID/", s.withRoom(s.postRoom))
	r.GET("/rooms/:roomID/updates/", s.withRoom(s.getRoomUpdates))

	s.Server.Handler = withReqID(withForm(r))

	return s
}

type Server struct {
	*http.Server
	db           *store.DB
	sessionStore *sessions.CookieStore
}

func (s *Server) static(basename string) httprouter.Handle {
	name := "web/static/" + basename
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		http.ServeFile(w, r, name)
	}
}

func withForm(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			// TODO: nicer HTML errors here and everywhere else
			http.Error(w, fmt.Sprintf("cannot parse form: %v", err),
				http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) renderPage(
	w http.ResponseWriter, r *http.Request,
	tpl *template.Template, keyval ...interface{},
) {
	s.renderFragment(w, r, tpl, "", keyval...)
}

func (s *Server) renderFragment(
	w http.ResponseWriter, r *http.Request,
	tpl *template.Template, name string, keyval ...interface{},
) {
	userName, _ := s.userName(r) // may be empty
	data := map[string]interface{}{
		"User": userName,
		"URL":  r.URL,
	}
	for i := 0; i < len(keyval); i += 2 {
		key := keyval[i].(string)
		val := keyval[i+1]
		data[key] = val
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var err error
	if name == "" {
		err = tpl.Execute(w, data)
	} else {
		err = tpl.ExecuteTemplate(w, name, data)
	}
	if err != nil {
		reqLogf(r, "failed to render HTML: %v", err)
		// Can't send 500 (Internal Server Error) here because
		// 200 (OK) may have already been sent.
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
