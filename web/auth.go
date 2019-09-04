package web

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _, ok := r.BasicAuth()
		if ok && user != "" {
			r = r.WithContext(context.WithValue(r.Context(), userKey, user))
		}
		next.ServeHTTP(w, r)
	})
}

func needAuth(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		_, ok := r.Context().Value(userKey).(string)
		if !ok {
			w.Header().Set("Www-Authenticate", `Basic realm="chatter"`)
			http.Error(w, "need authentication", http.StatusUnauthorized)
			return
		}
		next(w, r, ps)
	}
}

func mustUser(r *http.Request) string {
	return r.Context().Value(userKey).(string)
}
