package controller

import (
	"time"

	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
	"io"
	"k8s.io/helm/pkg/repo"
)

type RepoController struct {
	repos               []*repo.Entry
	repoChartMap        map[string]map[string]repo.ChartVersions
	repoChartUpdatedMap map[string]time.Time
	cacheLifetime       time.Duration
}

func NewRepoController(repos []*repo.Entry, cacheLifetime time.Duration) *RepoController {
	return &RepoController{
		repos:               repos,
		repoChartMap:        map[string]map[string]repo.ChartVersions{},
		repoChartUpdatedMap: map[string]time.Time{},
		cacheLifetime:       cacheLifetime,
	}
}

func (rc *RepoController) ListRepos() []*repo.Entry {
	return rc.repos
}

func (rc *RepoController) ListCharts(repoName string) (charts map[string]repo.ChartVersions, err error) {
	var repoURL string
	for _, repo := range rc.repos {
		if repo.Name == repoName {
			repoURL = repo.URL
			break
		}
	}
	// is it cached
	charts, found := rc.repoChartMap[repoName]
	if !found {
		// initial fetch if not
		log.Info("fetching initial chart list")
		charts, err = fetchCharts(repoURL)
		if err != nil {
			return
		}
		rc.repoChartMap[repoName] = charts
	}
	lastUpdated, found := rc.repoChartUpdatedMap[repoName]
	if !found {
		// set last updated
		lastUpdated = time.Now()
		rc.repoChartUpdatedMap[repoName] = lastUpdated
	}
	cacheLife := time.Now().Sub(lastUpdated)
	// is it expired
	if cacheLife >= rc.cacheLifetime {
		// update charts if it is
		log.Info("updating chart list")
		charts, err = fetchCharts(repoURL)
		if err != nil {
			return
		}
		rc.repoChartMap[repoName] = charts
		rc.repoChartUpdatedMap[repoName] = time.Now()
	}
	return
}

func (rc *RepoController) ChartDetails(repoName, chartName, chartVersion string) {
	// update charts if needed
	charts, err := rc.ListCharts(repoName)
	if err != nil {
		// unable to get charts
		return
	}
	versions := charts[chartName]
	var chartURLs []string
	for _, version := range versions {
		if version.Version == chartVersion {
			chartURLs = version.URLs
			break
		}
	}
	if chartURLs == nil {
		// version not found
		return
	}
	for _, chartURL := range chartURLs {
		fetchTarball(chartURL)
	}
	// loop through the URL, download, untar, create data struct, then return.
}

func fetchCharts(repoURL string) (map[string]repo.ChartVersions, error) {
	indexURL := repoURL + "/index.yaml"
	indexRes, err := http.Get(indexURL)
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
	return index.Entries, nil
}

func fetchTarball(url string) error {
	res, err := http.Get(url)
	if err != nil {
		// log pls
		return err
	}
	defer res.Body.Close()
	gzr, err := gzip.NewReader(res.Body)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			// end of file
			break
		}
		if err != nil {
			// something went wrong
		}
		log.Infof("tar: %v -- %v", header.Typeflag, header.Name)
		log.Info("-----")
		out, _ := ioutil.ReadAll(tr)
		log.Info(string(out))
		log.Info("-----")
	}
	// cool. got everything. just need to serve it.
	return nil
}
