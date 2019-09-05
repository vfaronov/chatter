package web

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/chatter/store"
)

var (
	signupTpl = template.Must(template.ParseFiles(
		"web/templates/page.html",
		"web/templates/signup.html",
	))
)

func (s *Server) getSignup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("bad query string: %v", err),
			http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := signupTpl.Execute(w, struct {
		Redir string
	}{
		Redir: r.Form.Get("redir"),
	})
	if err != nil {
		reqLogf(r, "failed to render page: %v", err)
	}
}

func (s *Server) postSignup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("bad form: %v", err), http.StatusBadRequest)
		return
	}

	user := &store.User{
		Name:     r.Form.Get("name"),
		Password: r.Form.Get("password"),
	}
	if user.Name == "" || user.Password == "" {
		http.Error(w, fmt.Sprintf("form must contain name and password"),
			http.StatusUnprocessableEntity)
		return
	}

	session, err := s.sess.Get(r, "session")
	if err != nil {
		http.Error(w, "cannot read session", http.StatusBadRequest)
		return
	}

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

	session.Values["name"] = user.Name
	err = session.Save(r, w)
	if err != nil {
		reqFatalf(w, r, err, "cannot save session")
		return
	}
	redir := r.Form.Get("redir")
	if redir == "" {
		redir = "/rooms/"
	}
	http.Redirect(w, r, redir, http.StatusSeeOther)
}
