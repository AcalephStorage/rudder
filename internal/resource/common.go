package resource

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
)

var (
	errFailToReadResponse  = restful.NewError(http.StatusBadRequest, "unable to read request body")
	errFailToWriteResponse = restful.NewError(http.StatusInternalServerError, "unable to write response")
)

// errorResponse creates an error response from the given error
func errorResponse(origErr error, res *restful.Response, err restful.ServiceError) {
	log.WithError(origErr).Error(err.Message)
	if err := res.WriteServiceError(err.Code, err); err != nil {
		log.WithError(origErr).Error("unable to write error")
	}
}
