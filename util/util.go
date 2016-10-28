package util

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
)

var ErrFailToWriteResponse = restful.NewError(http.StatusInternalServerError, "unable to write response")

func ErrorResponse(res *restful.Response, err restful.ServiceError) {
	log.WithError(err).Error(err.Message)
	if err := res.WriteServiceError(err.Code, err); err != nil {
		log.WithError(err).Error("unable to write error")
	}
}
