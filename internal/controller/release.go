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

// ReleaseController handles helm release related operations
type ReleaseController struct {
	tillerClient   *client.TillerClient
	repoController *RepoController
}

// NewReleaseController creates a new Release controller
func NewReleaseController(tillerClient *client.TillerClient, repoController *RepoController) *ReleaseController {
	return &ReleaseController{
		tillerClient:   tillerClient,
		repoController: repoController,
	}
}

// ListReleases returns a list of releases
func (rc *ReleaseController) ListReleases(req *tiller.ListReleasesRequest) (*tiller.ListReleasesResponse, error) {
	res, err := rc.tillerClient.ListReleases(req)
	if err != nil {
		log.WithError(err).Error("unable to get list of releases from tiller")
		return nil, err
	}
	return res, nil
}

// InstallRelease installs a new release of the provided chart
func (rc *ReleaseController) InstallRelease(name, namespace, repo, chart, version string, values map[string]interface{}) (*tiller.InstallReleaseResponse, error) {
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
		Name:      name,
		Namespace: namespace,
		Chart:     inChart,
		Values:    config,
	}

	res, err := rc.tillerClient.InstallRelease(req)
	if err != nil {
		log.WithError(err).Error("unable to install new release")
		return nil, err
	}
	return res, nil
}

// UpdateRelease updates a release with the provided chart
func (rc *ReleaseController) UpdateRelease(name, repo, chart, version string, values map[string]interface{}) (*tiller.UpdateReleaseResponse, error) {
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

	req := &tiller.UpdateReleaseRequest{
		Name:   name,
		Chart:  inChart,
		Values: config,
	}

	res, err := rc.tillerClient.UpdateRelease(req)
	if err != nil {
		log.WithError(err).Error("unable to update release")
		return nil, err
	}
	return res, nil
}

// UninstallRelease uninstall a release
func (rc *ReleaseController) UninstallRelease(releaseName string, purge bool) (*tiller.UninstallReleaseResponse, error) {
	req := &tiller.UninstallReleaseRequest{
		Name:  releaseName,
		Purge: purge,
	}

	res, err := rc.tillerClient.UninstallRelease(req)
	if err != nil {
		log.WithError(err).Error("unable to uninstall release")
		return nil, err
	}
	return res, nil
}

// GetRelease returns the release details
func (rc *ReleaseController) GetRelease(name string, version int32) (*tiller.GetReleaseContentResponse, error) {
	req := &tiller.GetReleaseContentRequest{
		Name:    name,
		Version: version,
	}
	content, err := rc.tillerClient.GetReleaseContent(req)
	if err != nil {
		log.WithError(err).Error("unable to get release content")
		return nil, err
	}
	return content, nil
}

// GetReleaseStatus returns the release status
func (rc *ReleaseController) GetReleaseStatus(name string, version int32) (*tiller.GetReleaseStatusResponse, error) {
	req := &tiller.GetReleaseStatusRequest{
		Name:    name,
		Version: version,
	}
	status, err := rc.tillerClient.GetReleaseStatus(req)
	if err != nil {
		log.WithError(err).Error("unable to get release status")
		return nil, err
	}
	return status, nil
}
