package resource

import (
	"strconv"
	"strings"

	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"k8s.io/helm/pkg/proto/hapi/release"
	tiller "k8s.io/helm/pkg/proto/hapi/services"

	"github.com/AcalephStorage/rudder/internal/controller"
	"github.com/AcalephStorage/rudder/internal/util"
)

var (
	sortByMap = map[string]tiller.ListSort_SortBy{
		"unknown":       tiller.ListSort_UNKNOWN,
		"name":          tiller.ListSort_NAME,
		"last-released": tiller.ListSort_LAST_RELEASED,
	}
	sortOrderMap = map[string]tiller.ListSort_SortOrder{
		"asc":  tiller.ListSort_ASC,
		"desc": tiller.ListSort_DESC,
	}
	statusCodeMap = map[string]release.Status_Code{
		"unknown":    release.Status_UNKNOWN,
		"deployed":   release.Status_DEPLOYED,
		"deleted":    release.Status_DELETED,
		"superseded": release.Status_SUPERSEDED,
		"failed":     release.Status_FAILED,
	}
)

var (
	errFailToListReleases      = restful.NewError(http.StatusBadRequest, "unable to get list of releases")
	errFailToInstallRelease    = restful.NewError(http.StatusInternalServerError, "unable to install releases")
	errFailtToUninstallRelease = restful.NewError(http.StatusInternalServerError, "unable to uninstall releases")
)

type InstallReleaseRequest struct {
	Repo      string                 `json:"repo"`
	Chart     string                 `json:"chart"`
	Version   string                 `json:"version"`
	Namespace string                 `json:"namespace"`
	Values    map[string]interface{} `json:"values"`
}

type ReleaseResource struct {
	controller *controller.ReleaseController
}

func NewReleaseResource(controller *controller.ReleaseController) *ReleaseResource {
	return &ReleaseResource{controller: controller}
}

func (rr *ReleaseResource) Register(container *restful.Container) {

	ws := new(restful.WebService)
	ws.Path("/api/v1/releases").
		Doc("Helm releases").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("").To(rr.listReleases).
		Doc("list releases").
		Operation("listReleases").
		Param(ws.QueryParameter("limit", "max number of releases to return")).
		Param(ws.QueryParameter("offset", "last release name that was seen")).
		Param(ws.QueryParameter("sort-by", "sort by: unknown, name, last-released")).
		Param(ws.QueryParameter("filter", "regex to filter releases")).
		Param(ws.QueryParameter("sort-order", "sort order: asc, desc")).
		Param(ws.QueryParameter("status-code", "comma-separated status codes: unknown, deployed, deleted, superseded, failed")).
		Writes(tiller.ListReleasesResponse{}))
	log.Debug("listReleases registered.")

	ws.Route(ws.POST("").To(rr.installRelease).
		Doc("install release").
		Operation("installRelease").
		Reads(InstallReleaseRequest{}).
		Writes(tiller.InstallReleaseResponse{}))

	ws.Route(ws.DELETE("/{release}").To(rr.uninstallRelease).
		Doc("uninstall release").
		Operation("uninstallRelease").
		Param(ws.PathParameter("release", "the release name to be deleted")))

	container.Add(ws)

}

// GET  api/v1/releases
func (rr *ReleaseResource) listReleases(req *restful.Request, res *restful.Response) {
	log.Info("Getting list of releases...")

	limit, _ := strconv.ParseInt(req.QueryParameter("limit"), 10, 64)
	offset := req.QueryParameter("offset")
	sortBy := sortByMap[req.QueryParameter("sort-by")]
	filter := req.QueryParameter("filter")
	sortOrder := sortOrderMap[req.QueryParameter("sort-order")]
	statusCodesRaw := req.QueryParameter("status-code")
	var statusCodes []release.Status_Code
	if len(statusCodesRaw) > 0 {
		scs := strings.Split(statusCodesRaw, ",")
		if len(scs) > 0 {
			statusCodes = make([]release.Status_Code, len(scs))
			for i, s := range scs {
				sc, ok := statusCodeMap[s]
				if ok {
					statusCodes[i] = sc
				}
			}
		}
	}

	request := &tiller.ListReleasesRequest{
		Limit:       limit,
		Offset:      offset,
		SortBy:      sortBy,
		Filter:      filter,
		SortOrder:   sortOrder,
		StatusCodes: statusCodes,
	}

	response, err := rr.controller.ListReleases(request)
	if err != nil {
		util.ErrorResponse(res, errFailToListReleases)
		return
	}
	if err := res.WriteEntity(response); err != nil {
		util.ErrorResponse(res, util.ErrFailToWriteResponse)
	}
}

// POST api/v1/releases {request body passed}
func (rr *ReleaseResource) installRelease(req *restful.Request, res *restful.Response) {
	var in InstallReleaseRequest
	if err := req.ReadEntity(&in); err != nil {
		util.ErrorResponse(res, util.ErrFailToReadResponse)
		return
	}
	out, err := rr.controller.InstallRelease(in.Repo, in.Chart, in.Version, in.Namespace, in.Values)
	if err != nil {
		util.ErrorResponse(res, errFailToInstallRelease)
		return
	}
	if err := res.WriteEntity(out); err != nil {
		util.ErrorResponse(res, util.ErrFailToWriteResponse)
	}
}

// DELETE api/v1/releases/:name {create request body}
func (rr *ReleaseResource) uninstallRelease(req *restful.Request, res *restful.Response) {
	releaseName := req.PathParameter("release")
	out, err := rr.controller.UninstallRelease(releaseName)
	if err != nil {
		util.ErrorResponse(res, errFailtToUninstallRelease)
		return
	}
	if err := res.WriteEntity(out); err != nil {
		util.ErrorResponse(res, util.ErrFailToWriteResponse)
	}
}

// GET api/v1/releases/:name/:version/:status {create request body}
func (rr *ReleaseResource) releaseStatus(req *restful.Request, res *restful.Response) {

}

// PUT api/v1/releases {request body passed}
func (rr *ReleaseResource) updateRelease(req *restful.Request, res *restful.Response) {

}

// POST ??? I DUNNOT KNOW
func (rr *ReleaseResource) rollbackRelease(req *restful.Request, res *restful.Response) {

}

// GET /api/v1/releases/:name/:version {create request body}
func (rr *ReleaseResource) releaseContent(req *restful.Request, res *restful.Response) {

}

// ???? I DUNNOT KNOW
func (rr *ReleaseResource) releaseHistory(req *restful.Request, res *restful.Response) {

}

// GET api/v1/version (do we need this?)
func (rr *ReleaseResource) getVersion(req *restful.Request, res *restful.Response) {

}
