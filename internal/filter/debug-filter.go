package filter

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
)

type DebugFilter struct{}

func NewDebugFilter() *DebugFilter {
	return &DebugFilter{}
}

func (df *DebugFilter) Debug(req *restful.Request, res *restful.Response, chain *restful.FilterChain) {
	log.Debugf("Request: Method=%v URL=%v", req.Request.Method, req.Request.URL)
	reqTime := time.Now()
	chain.ProcessFilter(req, res)
	resTime := time.Now()
	dur := resTime.Sub(reqTime)
	log.Debugf("Response: time=%v code=%v", dur.String(), res.StatusCode())
}
