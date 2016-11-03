package controller

import (
	log "github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/chartutil"
	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
	tiller "k8s.io/helm/pkg/proto/hapi/services"

	"fmt"
	"github.com/AcalephStorage/rudder/internal/client"
)

type ReleaseController struct {
	tillerClient   *client.TillerClient
	repoController *RepoController
}

func NewReleaseController(tillerClient *client.TillerClient, repoController *RepoController) *ReleaseController {
	return &ReleaseController{
		tillerClient:   tillerClient,
		repoController: repoController,
	}
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
	chartDetails, err := rc.repoController.ChartDetails(repo, chart, version)
	if err != nil {
		log.WithError(err).Error("unable to get chart details")
		return nil, err
	}
	tarball := chartDetails.ChartFile

	inChart, err := chartutil.LoadFile(tarball)
	if err != nil {
		log.WithError(err).Error("unable to load chart details")
		return nil, err
	}
	raw, _ := yaml.Marshal(values)

	inValues := make(map[string]*hapi_chart.Value)
	for k, v := range values {
		inValues[k] = &hapi_chart.Value{Value: fmt.Sprintf("%v", v)}
	}

	config := &hapi_chart.Config{
		Raw:    string(raw),
		Values: inValues,
	}

	req := &tiller.InstallReleaseRequest{
		Chart:     inChart,
		Namespace: namespace,
		Values:    config,
	}

	res, err := rc.tillerClient.InstallRelease(req)
	if err != nil {
		log.WithError(err).Error("unable to install new release")
		return nil, err
	}
	return res, nil
}

func (rc *ReleaseController) UninstallRelease(releaseName string) (*tiller.UninstallReleaseResponse, error) {
	req := &tiller.UninstallReleaseRequest{
		Name: releaseName,
	}

	res, err := rc.tillerClient.UninstallRelease(req)
	if err != nil {
		log.WithError(err).Error("unable to uninstall release")
		return nil, err
	}
	return res, nil
}
