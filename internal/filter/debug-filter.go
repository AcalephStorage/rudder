package filter

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
)

// DebugFilter provides a go-restful filter for logging debugging information
type DebugFilter struct{}

// NewDebugFilter returns a debug filter that logs some info about the request and response
func NewDebugFilter() *DebugFilter {
	return &DebugFilter{}
}

// Debug is a filter for logging the request method, URL, response time and code
func (df *DebugFilter) Debug(req *restful.Request, res *restful.Response, chain *restful.FilterChain) {
	log.Debugf("Request: Method=%v URL=%v", req.Request.Method, req.Request.URL)
	reqTime := time.Now()
	chain.ProcessFilter(req, res)
	resTime := time.Now()
	dur := resTime.Sub(reqTime)
	log.Debugf("Response: time=%v code=%v", dur.String(), res.StatusCode())
}
