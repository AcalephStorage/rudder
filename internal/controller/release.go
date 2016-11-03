package controller

import (
	log "github.com/Sirupsen/logrus"
	tiller "k8s.io/helm/pkg/proto/hapi/services"

	"github.com/AcalephStorage/rudder/internal/client"
)

type ReleaseController struct {
	tillerClient *client.TillerClient
}

func NewReleaseController(tillerClient *client.TillerClient) *ReleaseController {
	return &ReleaseController{tillerClient: tillerClient}
}

func (rc *ReleaseController) ListReleases(req *tiller.ListReleasesRequest) (*tiller.ListReleasesResponse, error) {
	res, err := rc.tillerClient.ListReleases(req)
	if err != nil {
		log.WithError(err).Error("unable to get list of releases from tiller")
		return nil, err
	}
	return res, nil
}

func (rc *ReleaseController) InstallRelease(repo, chart, version, namespace string, values map[string]interface{}) (*tiller.InstallReleaseResponse, error) {
	// create request here
	res, err := rc.tillerClient.InstallRelease(nil)
	if err != nil {
		log.WithError(err).Error("unable to install new release")
		return nil, err
	}
	return res, nil
}
