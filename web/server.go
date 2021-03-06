package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"

	"github.com/gorilla/sessions"
	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/nnbb/store"
)

var (
	//go:embed templates static
	assets       embed.FS
	templates, _ = fs.Sub(assets, "templates")
	static, _    = fs.Sub(assets, "static")
)

func NewServer(addr string, db *store.DB, key []byte) *Server {
	s := &Server{
		Server:       &http.Server{Addr: addr},
		db:           db,
		sessionStore: sessions.NewCookieStore(key),
	}

	r := httprouter.New()
	r.ServeFiles("/static/*filepath", http.FS(static))
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
	tpl *template.Template, payload interface{},
) {
	s.renderFragment(w, r, tpl, "", payload)
}

func (s *Server) renderFragment(
	w http.ResponseWriter, r *http.Request,
	tpl *template.Template, name string, payload interface{},
) {
	userName, _ := s.userName(r) // may be empty
	data := struct {
		User string
		URL  *url.URL
		P    interface{}
	}{
		User: userName,
		URL:  r.URL,
		P:    payload,
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

func loadPageTemplate(name string) *template.Template {
	return template.Must(template.New("page.html").Funcs(funcMap).ParseFS(templates, "page.html", name))
}
