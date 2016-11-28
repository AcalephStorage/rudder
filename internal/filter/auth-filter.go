package filter

import (
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
)

type AuthFilter struct {
	AuthList   []Auth
	Exceptions []string
}

type Auth interface {
	Authorize(req *restful.Request) bool
}

func (af *AuthFilter) Filter(req *restful.Request, res *restful.Response, chain *restful.FilterChain) {
	uri := req.Request.URL.RequestURI()
	for _, exception := range af.Exceptions {
		if strings.HasPrefix(uri, exception) {
			chain.ProcessFilter(req, res)
			return
		}
	}
	var success bool
	for _, auth := range af.AuthList {
		success = success || auth.Authorize(req)
		if success {
			break
		}
	}
	if !success {
		log.Debug("Unauthorized request rejected")
		res.WriteErrorString(401, "401: Unauthorized")
		return
	}
	chain.ProcessFilter(req, res)
}
