package filter

import (
	"strings"

	"encoding/base64"

	"github.com/emicklei/go-restful"
)

type AuthFilter struct {
	encodedAuth string
}

func NewAuthFilter(username, password string) *AuthFilter {
	code := "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
	return &AuthFilter{encodedAuth: code}
}

func (baf *AuthFilter) BasicAuthentication(req *restful.Request, res *restful.Response, chain *restful.FilterChain) {
	uri := req.Request.URL.RequestURI()
	exceptions := []string{
		"/apidocs.json",
		"/swagger",
	}
	for _, exception := range exceptions {
		if strings.HasPrefix(uri, exception) {
			chain.ProcessFilter(req, res)
			return
		}
	}
	encoded := req.Request.Header.Get("Authorization")
	if baf.encodedAuth != encoded {
		res.WriteErrorString(401, "401: Unauthorized")
		return
	}
	chain.ProcessFilter(req, res)
}
