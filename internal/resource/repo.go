package resource

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"k8s.io/helm/pkg/repo"

	"github.com/AcalephStorage/rudder/internal/controller"
)

var (
	errFailToGetCharts      = restful.NewError(http.StatusBadRequest, "unable to fetch charts")
	errFailToListVersions   = restful.NewError(http.StatusBadRequest, "unable to fetch chart versions")
	errFailToGetChartDetail = restful.NewError(http.StatusBadRequest, "unable to fetch chart details")
)

// RepoResource represents helm repositories
type RepoResource struct {
	controller *controller.RepoController
}

// NewRepoResource creates a new RepoResource
func NewRepoResource(controller *controller.RepoController) *RepoResource {
	return &RepoResource{controller: controller}
}

// Register registers this resource to the provided container
func (rr *RepoResource) Register(container *restful.Container) {

	ws := new(restful.WebService)

	ws.Path("/api/v1/repo").
		Doc("Helm repositories").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	// GET /api/v1/repo
	ws.Route(ws.GET("").To(rr.listRepos).
		Doc("list repos").
		Operation("listRepos").
		Writes([]repo.Entry{}))

	// GET /api/v1/repo/{repo}/charts
	ws.Route(ws.GET("{repo}/charts").To(rr.listCharts).
		Doc("list charts").
		Operation("listCharts").
		Param(ws.PathParameter("repo", "the helm repository")).
		Param(ws.QueryParameter("filter", "filter for the charts")).
		Writes(map[string][]repo.ChartVersion{}))

	// GET /api/v1/repo/{repo}/charts/{chart}
	ws.Route(ws.GET("{repo}/charts/{chart}").To(rr.listVersions).
		Doc("list chart versions").
		Operation("listVersions").
		Param(ws.PathParameter("repo", "the helm repository")).
		Param(ws.PathParameter("chart", "the helm chart")).
		Writes([]repo.ChartVersion{}))

	// GET /api/v1/repo/{repo}/charts/{chart}/{version}
	ws.Route(ws.GET("{repo}/charts/{chart}/{version}").To(rr.getChart).
		Doc("get chart details. specifying version=latest will return the chart tagged latest, or the top version if none is found").
		Operation("getChart").
		Param(ws.PathParameter("repo", "the helm repository")).
		Param(ws.PathParameter("chart", "the helm chart")).
		Param(ws.PathParameter("version", "the helm chart version")).
		Writes(controller.ChartDetail{}))

	container.Add(ws)
}

// listRepos returns a list of helm repositories taken from the repo file
func (rr *RepoResource) listRepos(req *restful.Request, res *restful.Response) {
	log.Info("Getting list of helm repositories...")
	repos := rr.controller.ListRepos()
	if err := res.WriteEntity(repos); err != nil {
		errorResponse(res, errFailToWriteResponse)
	}
}

// listCharts returns a list of charts from a repository
func (rr *RepoResource) listCharts(req *restful.Request, res *restful.Response) {
	repoName := req.PathParameter("repo")
	filter := req.QueryParameter("filter")

	charts, err := rr.controller.ListCharts(repoName, filter)
	if err != nil {
		errorResponse(res, errFailToGetCharts)
		return
	}
	// output
	if err := res.WriteEntity(charts); err != nil {
		errorResponse(res, errFailToWriteResponse)
	}
}

// listVersions returns a list of chart versions
func (rr *RepoResource) listVersions(req *restful.Request, res *restful.Response) {
	repoName := req.PathParameter("repo")
	chartName := req.PathParameter("chart")

	charts, err := rr.controller.ListCharts(repoName, chartName)
	if err != nil {
		errorResponse(res, errFailToListVersions)
		return
	}

	versions := charts[chartName]
	if err := res.WriteEntity(versions); err != nil {
		errorResponse(res, errFailToWriteResponse)
	}

}

// getChart returns the chart details
func (rr *RepoResource) getChart(req *restful.Request, res *restful.Response) {
	repoName := req.PathParameter("repo")
	chartName := req.PathParameter("chart")
	chartVersion := req.PathParameter("version")

	chartDetail, err := rr.controller.ChartDetails(repoName, chartName, chartVersion)
	if err != nil {
		errorResponse(res, errFailToGetChartDetail)
		return
	}

	if err := res.WriteEntity(chartDetail); err != nil {
		errorResponse(res, errFailToWriteResponse)
	}
}
