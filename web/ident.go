package web

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

// withReqID is a middleware that assigns a random ID to the request's context,
// so that log lines pertaining to it can be correlated, and also logs the request.
// Headers like X-Request-ID are not considered, because nnBB is supposed to be
// a user-facing service.
func withReqID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b [8]byte
		rand.Read(b[:])
		reqID := base64.RawURLEncoding.EncodeToString(b[:])
		log.Printf("[%v] %v %v from %v", reqID, r.Method, r.URL, briefUserAgent(r))
		r = r.WithContext(context.WithValue(r.Context(), reqIDKey, reqID))
		next.ServeHTTP(w, r)
	})
}

func reqLogf(r *http.Request, format string, v ...interface{}) {
	reqID := r.Context().Value(reqIDKey).(string)
	log.Print("[", reqID, "] ", fmt.Sprintf(format, v...))
}

func reqFatalf(w http.ResponseWriter, r *http.Request, err error, format string, v ...interface{}) {
	reqID := r.Context().Value(reqIDKey).(string)
	msg := fmt.Sprintf(format, v...)
	log.Print("[", reqID, "] ", msg, ": ", err)
	http.Error(w, msg, http.StatusInternalServerError)
}

// briefUserAgent returns the first product from the User-Agent header of r.
func briefUserAgent(r *http.Request) string {
	ua := r.Header.Get("User-Agent")
	if ua == "" {
		return "unknown"
	}
	pos := strings.IndexByte(ua, ' ')
	if pos == -1 {
		return ua
	}
	return ua[:pos]
}
