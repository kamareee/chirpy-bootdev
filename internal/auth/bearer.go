package auth

import (
	"net/http"
	"strings"
)

func GetBearerToken(headers http.Header) (string, error) {
	bearerStr := strings.Split(headers.Get("Authorization"), " ")
	if len(bearerStr) != 2 || bearerStr[0] != "Bearer" {
		return "", http.ErrNoCookie
	}
	return bearerStr[1], nil
}
