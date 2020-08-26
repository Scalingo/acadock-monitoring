package handlers

import (
	"bufio"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
)

func AuthMiddleware(check func(user, password string) bool) MiddlewareFunc {
	return MiddlewareFunc(func(handler HandlerFunc) HandlerFunc {
		return func(res http.ResponseWriter, req *http.Request, vars map[string]string) error {
			auth := req.Header.Get("Authorization")
			if auth == "" {
				res.WriteHeader(401)
				return errors.New("no authentication provided")
			}

			b64auth := strings.Split(auth, " ")[1]
			authReader := strings.NewReader(b64auth)
			dec := base64.NewDecoder(base64.StdEncoding, authReader)
			bufioReader := bufio.NewReader(dec)
			clearAuth, err := bufioReader.ReadString('\x00')
			if err != nil && err != io.EOF {
				res.WriteHeader(401)
				return err
			}
			clearAuthArr := strings.SplitN(clearAuth, ":", 2)
			httpUser, httpPassword := clearAuthArr[0], clearAuthArr[1]

			if !check(httpUser, httpPassword) {
				res.WriteHeader(401)
				return errors.New("invalid auth")
			}
			return handler(res, req, vars)
		}
	})
}
