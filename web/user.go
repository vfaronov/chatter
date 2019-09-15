package web

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/chatter/store"
)

var (
	signupTpl = template.Must(template.ParseFiles(
		"web/templates/page.html",
		"web/templates/signup.html",
	))
)

func (s *Server) session(r *http.Request) *sessions.Session {
	// Gorilla's docs suggest checking error and responding with 500;
	// but an invalid session should not abort handling (much less with a 500),
	// it should just be ignored, creating a new session.
	sess, _ := s.sessionStore.Get(r, "session")
	return sess
}

func (s *Server) userName(r *http.Request) (string, bool) {
	name, ok := s.session(r).Values["name"].(string)
	return name, ok
}

func (s *Server) getSignup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	s.renderPage(w, r, signupTpl, "Redir", r.Form.Get("redir"))
}

func (s *Server) postSignup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()
	user := &store.User{
		Name:     r.Form.Get("name"),
		Password: r.Form.Get("password"),
	}
	if user.Name == "" || user.Password == "" {
		http.Error(w, fmt.Sprintf("form must contain name and password"),
			http.StatusUnprocessableEntity)
		return
	}

	var err error
	switch r.Form.Get("action") {
	case "sign-up":
		reqLogf(r, "sign up %v", user.Name)
		err = s.db.CreateUser(ctx, user)
	case "log-in":
		reqLogf(r, "log in %v", user.Name)
		err = s.db.Authenticate(ctx, user)
	default:
		http.Error(w, fmt.Sprintf("bad action %q", r.Form.Get("action")),
			http.StatusUnprocessableEntity)
		return
	}
	switch err {
	case store.ErrBadCredentials, store.ErrDuplicate:
		reqLogf(r, err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	case nil:
		// OK
	default:
		reqFatalf(w, r, err, "failed to %v", r.Form.Get("action"))
		return
	}

	sess := s.session(r)
	sess.Values["name"] = user.Name
	err = sess.Save(r, w)
	if err != nil {
		reqFatalf(w, r, err, "cannot save session")
		return
	}
	redir := r.Form.Get("redir") // TODO: use Referer instead (here and elsewhere)
	if redir == "" {
		redir = "/rooms/"
	}
	http.Redirect(w, r, redir, http.StatusSeeOther)
}

func (s *Server) postLogout(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	sess := s.session(r)
	if sess.Options == nil {
		sess.Options = &sessions.Options{}
	}
	sess.Options.MaxAge = -1
	if err := sess.Save(r, w); err != nil {
		reqFatalf(w, r, err, "failed to save session")
		return
	}
	redir := r.Form.Get("redir")
	if redir == "" {
		redir = "/rooms/"
	}
	http.Redirect(w, r, redir, http.StatusSeeOther)
}
