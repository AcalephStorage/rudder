package resource

import (
	"net/http"

	"github.com/AcalephStorage/rudder/client"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"k8s.io/helm/pkg/proto/hapi/release"
	tiller "k8s.io/helm/pkg/proto/hapi/services"
	"strconv"
	"strings"
)

type ReleaseResource struct {
	tillerClient *client.TillerClient
}

func NewReleaseResource(tillerClient *client.TillerClient) *ReleaseResource {
	return &ReleaseResource{tillerClient: tillerClient}
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
		Param(ws.QueryParameter("sort-order", "0 asc, 1 desc")).
		Param(ws.QueryParameter("status-code", "0unknown, 1deployed, 2deleted 3superseded 4 failed")).
		Writes(tiller.ListReleasesResponse{}))
	log.Debug("listReleases registered.")

	ws.Route(ws.POST("").To(rr.installRelease).
		Doc("install release").
		Operation("installRelease"))

	container.Add(ws)

	// ws.Route(ws.DELETE("/{release}").To(rr.deleteRelease).
	// 	Doc("delete release").
	// 	Operation("deleteRelease"))

	// ws.Route(ws)
}

// GET  api/v1/releases
func (rr *ReleaseResource) listReleases(req *restful.Request, res *restful.Response) {
	log.Info("Getting list of releases...")

	request := &tiller.ListReleasesRequest{}
	// set limit
	limit, err := strconv.ParseInt(req.QueryParameter("limit"), 10, 64)
	if err == nil {
		request.Limit = limit
	}
	// offset
	offset := req.QueryParameter("offset")
	if offset != "" {
		request.Offset = offset
	}
	// sortBy
	sortByRaw := req.QueryParameter("sort-by")
	sortByMap := map[string]tiller.ListSort_SortBy{
		"unknown":       tiller.ListSort_UNKNOWN,
		"name":          tiller.ListSort_NAME,
		"last-released": tiller.ListSort_LAST_RELEASED,
	}
	sortBy, ok := sortByMap[sortByRaw]
	if ok {
		request.SortBy = sortBy
	}
	// filter
	filter := req.QueryParameter("filter")
	if filter != "" {
		request.Filter = filter
	}
	// statusCode
	statusCodesRaw := req.QueryParameter("status-code")
	statusCodeMap := map[string]release.Status_Code{
		"unknown":    release.Status_UNKNOWN,
		"deployed":   release.Status_DEPLOYED,
		"deleted":    release.Status_DELETED,
		"superseded": release.Status_SUPERSEDED,
		"failed":     release.Status_FAILED,
	}
	if statusCodesRaw != "" {
		scs := strings.Split(statusCodesRaw, ",")
		statusCodes := make([]release.Status_Code, len(scs))
		for i, s := range scs {
			sc, ok := statusCodeMap[s]
			if ok {
				statusCodes[i] = sc
			}
		}
		if len(statusCodes) > 0 {
			request.StatusCodes = statusCodes
		}
	}

	response, err := rr.tillerClient.ListReleases(request)
	if err != nil {
		log.WithError(err).Error("unable to get list releases response from tiller")
		res.WriteErrorString(http.StatusInternalServerError, err.Error())
		return
	}
	if err := res.WriteEntity(response); err != nil {
		log.WithError(err).Error("unable to write list releases response")
		res.WriteErrorString(http.StatusInternalServerError, err.Error())
	}
}

// POST api/v1/releases {request body passed}
func (rr *ReleaseResource) installRelease(req *restful.Request, res *restful.Response) {

}

// DELETE api/v1/releases/:name?disable-hooks&purge {create request body}
func (rr *ReleaseResource) deleteRelease(req *restful.Request, res *restful.Response) {

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
