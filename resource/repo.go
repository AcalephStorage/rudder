package resource

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"k8s.io/helm/pkg/repo"

	"github.com/AcalephStorage/rudder/controller"
	"github.com/AcalephStorage/rudder/util"
)

var (
	errFailToGetCharts      = restful.NewError(http.StatusBadRequest, "unable to fetch charts")
	errFailToGetChartDetail = restful.NewError(http.StatusBadRequest, "unable to fetch chart details")
)

type RepoResource struct {
	controller *controller.RepoController
}

func NewRepoResource(controller *controller.RepoController) *RepoResource {
	return &RepoResource{controller: controller}
}

func (rr *RepoResource) Register(container *restful.Container) {

	ws := new(restful.WebService)

	ws.Path("/api/v1/repo").
		Doc("Helm repositories").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("").To(rr.listRepos).
		Doc("list repos").
		Operation("listRepos").
		Writes([]repo.Entry{}))

	ws.Route(ws.GET("{repo}/charts").To(rr.listCharts).
		Doc("list charts").
		Operation("listCharts").
		Param(ws.PathParameter("repo", "the helm repository")).
		Writes(map[string][]repo.ChartVersion{}))
	log.Debug("listCharts registered.")

	ws.Route(ws.GET("{repo}/charts/{chart}/{version}").To(rr.getChart).
		Doc("get chart").
		Operation("getChart").
		Param(ws.PathParameter("repo", "the helm repository")).
		Param(ws.PathParameter("chart", "the helm chart")).
		Param(ws.PathParameter("version", "the helm chart version")).
		Writes(controller.ChartDetail{}))

	container.Add(ws)
}

func (rr *RepoResource) listRepos(req *restful.Request, res *restful.Response) {
	log.Info("Getting list of helm repositories...")
	repos := rr.controller.ListRepos()
	if err := res.WriteEntity(repos); err != nil {
		util.ErrorResponse(res, util.ErrFailToWriteResponse)
	}
}

func (rr *RepoResource) listCharts(req *restful.Request, res *restful.Response) {
	repoName := req.PathParameter("repo")

	charts, err := rr.controller.ListCharts(repoName)
	if err != nil {
		util.ErrorResponse(res, errFailToGetCharts)
		return
	}
	// output
	if err := res.WriteEntity(charts); err != nil {
		util.ErrorResponse(res, util.ErrFailToWriteResponse)
	}
}

func (rr *RepoResource) getChart(req *restful.Request, res *restful.Response) {
	repoName := req.PathParameter("repo")
	chartName := req.PathParameter("chart")
	chartVersion := req.PathParameter("version")

	chartDetail, err := rr.controller.ChartDetails(repoName, chartName, chartVersion)
	if err != nil {
		util.ErrorResponse(res, errFailToGetChartDetail)
		return
	}

	if err := res.WriteEntity(chartDetail); err != nil {
		util.ErrorResponse(res, util.ErrFailToWriteResponse)
	}
	// return whatever is returned from this
}
