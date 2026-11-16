package routes

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/aosanya/mwanachama-backend-shared/httpwire"
)

const unreadableQueryMessage = "the query string could not be read: too many parameters"

const probeSegments = 1 << 20

var queryCapError = func() error {
	_, err := url.ParseQuery(strings.Repeat("&", probeSegments))
	return err
}()

func IsQueryCapError(err error) bool {
	return err != nil && queryCapError != nil && err.Error() == queryCapError.Error()
}

func refuseUnreadableQuery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := url.ParseQuery(r.URL.RawQuery); IsQueryCapError(err) {
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
