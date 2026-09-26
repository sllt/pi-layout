package middleware

import (
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/sllt/pi-layout/pkg/errcode"
	"github.com/sllt/pi-layout/pkg/log"
)

func SignMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	appKey := os.Getenv("API_SIGN_KEY")
	appSecret := os.Getenv("API_SIGN_SECRET")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requiredHeaders := []string{"Timestamp", "Nonce", "Sign", "App-Version"}

			for _, header := range requiredHeaders {
				value := r.Header.Get(header)
				if value == "" {
					if logger != nil {
						logger.Warnf("missing signature header=%s url=%s", header, r.URL)
					}
					errcode.WriteHTTPError(w, r, errcode.ErrBadRequest)
					return
				}
			}

			data := map[string]string{
				"AppKey":     appKey,
				"Timestamp":  r.Header.Get("Timestamp"),
				"Nonce":      r.Header.Get("Nonce"),
				"AppVersion": r.Header.Get("App-Version"),
			}

			var keys []string
			for k := range data {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return strings.ToLower(keys[i]) < strings.ToLower(keys[j]) })

			var str string
			for _, k := range keys {
				str += k + data[k]
			}
			str += appSecret

			if r.Header.Get("Sign") != strings.ToUpper(cryptor.Md5String(str)) {
				if logger != nil {
					logger.Warnf("invalid signature url=%s", r.URL)
				}
				errcode.WriteHTTPError(w, r, errcode.ErrInvalidSignature)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
