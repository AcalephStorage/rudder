package filter

import (
	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
)

type BasicAuth struct {
	Username string
	Password string
}

func (ba *BasicAuth) Authorize(req *restful.Request) (authorized bool) {
	log.Debug("verifying Basic Auth")
	username, password, ok := req.Request.BasicAuth()
	if ok && username == ba.Username && password == ba.Password {
		log.Debug("failed to verify using Basic Auth")
		authorized = true
	}
	return
}
