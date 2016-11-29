package filter

import (
	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
)

// BasicAuth provides basic authentication with username and password
type BasicAuth struct {
	username string
	password string
}

// NewBasicAuth creates a new BasicAuth authentication
func NewBasicAuth(username, password string) *BasicAuth {
	return &BasicAuth{username: username, password: password}
}

// Authorize returns true if the request contains the correct basic auth
func (ba *BasicAuth) Authorize(req *restful.Request) (authorized bool) {
	log.Debug("verifying Basic Auth")
	username, password, ok := req.Request.BasicAuth()
	if ok && username == ba.username && password == ba.password {
		log.Debug("failed to verify using Basic Auth")
		authorized = true
	}
	return
}
