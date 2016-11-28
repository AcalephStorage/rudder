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

	addressFlag                   = "address"
	tillerAddressFlag             = "tiller-address"
	helmRepoFileFlag              = "helm-repo-file"
	helmCacheDirFlag              = "helm-cache-dir"
	helmRepoCacheLifetimeFlag     = "helm-repo-cache-lifetime"
	swaggerUIPathFlag             = "swagger-ui-path"
	basicAuthUsernameFlag         = "basic-auth-username"
	basicAuthPasswordFlag         = "basic-auth-password"
	oidcIssuerURLFlag             = "oidc-issuer-url"
	clientIDFlag                  = "client-id"
	clientSecretFlag              = "client-secret"
	clientSecretBase64EncodedFlag = "client-secret-base64-encoded"
	debugFlag                     = "debug"
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
		cli.StringFlag{
			Name:   basicAuthUsernameFlag,
			Usage:  "basic auth username. will enable basic authentication if both username and password are provided",
			EnvVar: "RUDDER_BASIC_AUTH_USERNAME",
		},
		cli.StringFlag{
			Name:   basicAuthPasswordFlag,
			Usage:  "basic auth password. will enable basic authentication if both username and password are provided",
			EnvVar: "RUDDER_BASIC_AUTH_PASSWORD",
		},
		cli.StringFlag{
			Name:   oidcIssuerURLFlag,
			Usage:  "OIDC issuer url. will enable OIDC authentication",
			EnvVar: "RUDDER_OIDC_ISSUER_URL",
		},
		cli.StringFlag{
			Name:   "client-id",
			Usage:  "OAuth Client ID. if specified, oidc will verify 'aud'",
			EnvVar: "RUDDER_CLIENT_ID",
		},
		cli.StringFlag{
			Name:   "client-secret",
			Usage:  "OAuth Client Secret. if specified and JWT is signed with HMAC, this will be used for verification",
			EnvVar: "RUDDER_CLIENT_SECRET",
		},
		cli.BoolFlag{
			Name:   clientSecretBase64EncodedFlag,
			Usage:  "enable this flag to specify that the client-secret is base64 encoded",
			EnvVar: "RUDDER_CLIENT_BASE64_ENCODED",
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
	helmRepoFile := ctx.String(helmRepoFileFlag)
	helmCacheDir := ctx.String(helmCacheDirFlag)
	helmRepoCacheLifetime := ctx.Int(helmRepoCacheLifetimeFlag)
	swaggerUIPath := ctx.String(swaggerUIPathFlag)
	basicAuthUsername := ctx.String(basicAuthUsernameFlag)
	basicAuthPassword := ctx.String(basicAuthPasswordFlag)
	oidcIssuerURL := ctx.String(oidcIssuerURLFlag)
	clientID := ctx.String(clientIDFlag)
	clientSecret := ctx.String(clientSecretFlag)
	clientSecretBase64Encoded := ctx.Bool(clientSecretBase64EncodedFlag)
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

	// authList
	authList := make([]filter.Auth, 0)
	if len(basicAuthUsername) > 0 && len(basicAuthPassword) > 0 {
		authList = append(authList, &filter.BasicAuth{
			basicAuthUsername,
			basicAuthPassword,
		})
	}
	if len(oidcIssuerURL) > 0 || len(clientSecret) > 0 {
		oidcAuth, err := filter.NewOIDCAuth(oidcIssuerURL, clientID, clientSecret, clientSecretBase64Encoded)
		if err != nil {
			log.WithError(err).Error("unable to connect to OIDC issuer")
			return err
		}
		authList = append(authList, oidcAuth)
	}

	// auth filter
	authFilter := &filter.AuthFilter{
		AuthList: authList,
		Exceptions: []string{
			"/apidocs.json",
			"/swagger",
		},
	}
	container.Filter(authFilter.Filter)
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
