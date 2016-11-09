package main

import (
	"os"

	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	restfullog "github.com/emicklei/go-restful/log"
	"github.com/emicklei/go-restful/swagger"
	"github.com/ghodss/yaml"
	"github.com/urfave/cli"
	"k8s.io/helm/pkg/repo"

	"github.com/AcalephStorage/rudder/internal/client"
	"github.com/AcalephStorage/rudder/internal/controller"
	"github.com/AcalephStorage/rudder/internal/filter"
	"github.com/AcalephStorage/rudder/internal/resource"
	"io/ioutil"
	"time"
)

const (
	appName = "rudder"

	addressFlag               = "address"
	tillerAddressFlag         = "tiller-address"
	authUsernameFlag          = "auth-username"
	authPasswordFlag          = "auth-password"
	helmRepoFileFlag          = "helm-repo-file"
	helmCacheDirFlag          = "helm-cache-dir"
	helmRepoCacheLifetimeFlag = "helm-repo-cache-lifetime"
	swaggerUIPathFlag         = "swagger-ui-path"
	debugFlag                 = "debug"
)

var (
	version = "dev"
)

func main() {
	app := cli.NewApp()
	app.Name = appName
	app.Version = version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   addressFlag,
			Usage:  "bind address",
			EnvVar: "RUDDER_ADDRESS",
			Value:  "0.0.0.0:5000",
		},
		cli.StringFlag{
			Name:   tillerAddressFlag,
			Usage:  "tiller address",
			EnvVar: "RUDDER_TILLER_ADDRESS",
			Value:  "localhost:44134",
		},
		cli.StringFlag{
			Name:   authUsernameFlag,
			Usage:  "basic auth username",
			EnvVar: "RUDDER_AUTH_USERNAME",
			Value:  "admin",
		},
		cli.StringFlag{
			Name:   authPasswordFlag,
			Usage:  "basic auth password",
			EnvVar: "RUDDER_AUTH_PASSWORD",
			Value:  "admin",
		},
		cli.StringFlag{
			Name:   helmRepoFileFlag,
			Usage:  "helm repo file",
			EnvVar: "RUDDER_HELM_REPO_FILE",
			Value:  os.Getenv("HOME") + "/.helm/repository/repositories.yaml",
		},
		cli.StringFlag{
			Name:   helmCacheDirFlag,
			Usage:  "helm cache dir",
			EnvVar: "RUDDER_HELM_CACHE_DIR",
			Value:  "/opt/rudder/cache",
		},
		cli.IntFlag{
			Name:   helmRepoCacheLifetimeFlag,
			Usage:  "cache lifetime before it gets updated (mins)",
			EnvVar: "RUDDER_HELM_REPO_CACHE_LIFETIME",
			Value:  10,
		},
		cli.StringFlag{
			Name:   swaggerUIPathFlag,
			Usage:  "swagger ui path",
			EnvVar: "RUDDER_SWAGGER_UI_PATH",
			Value:  "/opt/rudder/swagger",
		},
		cli.BoolFlag{
			Name:   debugFlag,
			Hidden: true,
		},
	}

	app.Action = startRudder
	app.Run(os.Args)
}

func startRudder(ctx *cli.Context) error {
	// unified logging format
	logFormat := &log.TextFormatter{FullTimestamp: true}
	log.SetFormatter(logFormat)
	restfulLogger := log.New()
	restfulLogger.Formatter = logFormat
	restfullog.SetLogger(restfulLogger)

	log.Info("Starting Rudder...")

	// flags
	address := ctx.String(addressFlag)
	tillerAddress := ctx.String(tillerAddressFlag)
	authUsername := ctx.String(authUsernameFlag)
	authPassword := ctx.String(authPasswordFlag)
	helmRepoFile := ctx.String(helmRepoFileFlag)
	helmCacheDir := ctx.String(helmCacheDirFlag)
	helmRepoCacheLifetime := ctx.Int(helmRepoCacheLifetimeFlag)
	swaggerUIPath := ctx.String(swaggerUIPathFlag)
	isDebug := ctx.Bool(debugFlag)

	container := restful.NewContainer()

	// debug mode
	if isDebug {
		log.SetLevel(log.DebugLevel)
		log.Debug("DEBUG Mode enabled.")
		debugFilter := filter.NewDebugFilter()
		container.Filter(debugFilter.Debug)
		log.Debug("Debug filter added.")
	}

	// cors
	cors := restful.CrossOriginResourceSharing{
		AllowedMethods: []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowedHeaders: []string{"Authorization", "Accept", "Content-Type"},
		Container:      container,
	}
	container.Filter(cors.Filter)
	log.Info("CORS filter added.")

	// options
	container.Filter(container.OPTIONSFilter)
	log.Info("OPTIONS filter added.")

	// auth filter
	authFilter := filter.NewAuthFilter(authUsername, authPassword)
	container.Filter(authFilter.BasicAuthentication)
	log.Info("Auth filter added")

	// repo resource (TODO: refactor pls)
	repoFileYAML, err := ioutil.ReadFile(helmRepoFile)
	if err != nil {
		log.Fatalf("unable to read repo file at %s", helmRepoFile)
	}
	repoFileJSON, err := yaml.YAMLToJSON(repoFileYAML)
	if err != nil {
		log.Fatal("unable to convert yaml to json")
	}
	var repoFile repo.RepoFile
	if err := json.Unmarshal(repoFileJSON, &repoFile); err != nil {
		log.Fatal("unable to parse repoFile")
	}
	cacheDur := (time.Duration(helmRepoCacheLifetime) * time.Minute)
	repoController := controller.NewRepoController(repoFile.Repositories, helmCacheDir, cacheDur)
	repoResource := resource.NewRepoResource(repoController)
	repoResource.Register(container)
	log.Info("repo resource registered.")

	// releaseController
	tillerClient := client.NewTillerClient(tillerAddress)
	releaseController := controller.NewReleaseController(tillerClient, repoController)
	// release resource
	releaseResource := resource.NewReleaseResource(releaseController)
	releaseResource.Register(container)
	log.Info("release resource registered.")

	// enable swagger
	swaggerConfig := swagger.Config{
		WebServices: container.RegisteredWebServices(),
		ApiPath:     "/apidocs.json",
		ApiVersion:  version,
		Info: swagger.Info{
			Title:       "Rudder",
			Description: "RESTful proxy for the Tiller service",
		},
		SwaggerPath:     "/swagger/",
		SwaggerFilePath: swaggerUIPath,
	}
	swagger.RegisterSwaggerService(swaggerConfig, container)
	log.Info("Swagger enabled.")

	log.Infof("Rudder listening at: %v", address)
	return http.ListenAndServe(address, container)
}
