package resource

import (
	"io/ioutil"
	"net/http"

	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/repo"
)

type RepoResource struct {
	// refactor: should be moved to a service struct
	repoFile *repo.RepoFile
}

func NewRepoResource(repoFile *repo.RepoFile) *RepoResource {
	return &RepoResource{repoFile: repoFile}
}

func (rr *RepoResource) Register(container *restful.Container) {

	ws := new(restful.WebService)

	ws.Path("/api/v1/repositories").
		Doc("Helm repositories").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("{repo}/charts").To(rr.listCharts).
		Doc("list charts").
		Operation("listCharts").
		Param(ws.PathParameter("repo", "the helm repository")).
		Writes(repo.IndexFile{}))
	log.Debug("listCharts registered.")

	ws.Route(ws.GET("{repo}/charts/{chart}/{version}").To(rr.getChart).
		Doc("get chart").
		Operation("getChart").
		Param(ws.PathParameter("repo", "the helm repository")).
		Param(ws.PathParameter("chart", "the helm chart")).
		Param(ws.PathParameter("version", "the helm chart version")))

	container.Add(ws)
}

func (rr *RepoResource) listCharts(req *restful.Request, res *restful.Response) {
	repoName := req.PathParameter("repo")

	curRepo, found := findRepo(repoName, rr.repoFile)
	if !found {
		// repo not found
		log.Errorf("%s is not a defined repository", repoName)
		res.WriteErrorString(http.StatusBadRequest, "repository not defined")
		return
	}

	// index file url
	repoURL := curRepo.URL + "/index.yaml"
	index, err := readRepoIndex(repoURL)
	if err != nil {
		res.WriteErrorString(http.StatusBadRequest, "failed to load repository index")
		return
	}

	// output
	if err := res.WriteEntity(index); err != nil {
		log.WithError(err).Error("unable to write response entity")
		res.WriteErrorString(http.StatusInternalServerError, "unable to write response")
	}
}

func (rr *RepoResource) getChart(req *restful.Request, res *restful.Response) {
	repoName := req.PathParameter("repo")
	chartName := req.PathParameter("chart")
	chartVersion := req.PathParameter("version")

	curRepo, found := findRepo(repoName, rr.repoFile)
	if !found {
		// repo not found
		log.Errorf("%s is not a defined repository", repoName)
		res.WriteErrorString(http.StatusBadRequest, "repository not defined")
		return
	}

	// index file url
	repoURL := curRepo.URL + "/index.yaml"
	index, err := readRepoIndex(repoURL)
	if err != nil {
		res.WriteErrorString(http.StatusBadRequest, "failed to load repository index")
		return
	}

	chart, err := index.Get(chartName, chartVersion)
	if err != nil {
		log.WithError(err).Errorf("unable to find %s:%s chart", chartName, chartVersion)
		res.WriteErrorString(http.StatusBadRequest, "unable to find helm chart")
		return
	}
	if err := res.WriteEntity(chart); err != nil {
		log.WithError(err).Error("unable to write response entity")
		res.WriteErrorString(http.StatusInternalServerError, "unabler to write response")
	}
}

// refactor: extact to service package
func findRepo(repoName string, repoFile *repo.RepoFile) (*repo.Entry, bool) {
	for _, r := range repoFile.Repositories {
		if r.Name == repoName {
			return r, true
		}
	}
	return nil, false
}

// refactor: extact to service package
func readRepoIndex(repoURL string) (*repo.IndexFile, error) {
	indexRes, err := http.Get(repoURL)
	if err != nil {
		// unable to get index.yaml
		log.WithError(err).Errorf("unable to get repository index at %s", repoURL)
		return nil, err
	}
	body := indexRes.Body
	defer body.Close()
	indexYAML, err := ioutil.ReadAll(body)
	if err != nil {
		// unable to read index.yaml
		log.WithError(err).Error("unable to read index.yaml from response body")
		return nil, err
	}
	indexJSON, err := yaml.YAMLToJSON(indexYAML)
	if err != nil {
		log.WithError(err).Errorf("unable to convert index.yaml to JSON")
		return nil, err
	}
	var index repo.IndexFile
	if err := json.Unmarshal(indexJSON, &index); err != nil {
		log.WithError(err).Errorf("unable to unmarshal index JSON")
		return nil, err
	}
	return &index, nil
}
