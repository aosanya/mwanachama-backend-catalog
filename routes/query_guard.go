package routes

import (
	"net/http"
	"net/url"

	"github.com/aosanya/mwanachama-backend-shared/httpwire"
)

const unreadableQueryMessage = "the query string could not be read: too many parameters"

func refuseUnreadableQuery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := url.ParseQuery(r.URL.RawQuery); err != nil {
			httpwire.WriteErr(w, http.StatusBadRequest, unreadableQueryMessage)
			return
		}
		next(w, r)
	}
}

func guardQueries(rs []Route) []Route {
	for i := range rs {
		rs[i].Handler = refuseUnreadableQuery(rs[i].Handler)
	}
	return rs
}
