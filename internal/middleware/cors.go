
package middleware

import "net/http"

func CORS(next http.Handler) http.Handler {

	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {

		w.Header().Set(
			"Access-Control-Allow-Origin",
			"*",
		)

		w.Header().Set(
			"Access-Control-Allow-Methods",
			"GET, HEAD, OPTIONS",
		)

		w.Header().Set(
			"Access-Control-Allow-Headers",
			"Range, Origin, Accept",
		)

		w.Header().Set(
			"Cache-Control",
			"no-cache",
		)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
