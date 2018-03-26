package main

import (
	"os"
	"time"

	"encoding/json"
	"io/ioutil"
	"net/http"

	auth "github.com/AcalephStorage/go-auth"
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
	insecure                      = "insecure"

	swaggerAPIPath = "/apidocs.json"
	swaggerPath    = "/swagger/"
)

var (
	version = "dev"

	corsAllowedMethods = []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"}
	corsAllowedHeaders = []string{"Authorization", "Accept", "Content-Type"}
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
		cli.DurationFlag{
			Name:   helmRepoCacheLifetimeFlag,
			Usage:  "cache lifetime. should be in duration format (eg. 10m). valid time units are 'ns', 'us' (or 'µs'), 'ms', 's', 'm', 'h'",
			EnvVar: "RUDDER_HELM_REPO_CACHE_LIFETIME",
			Value:  10 * time.Minute,
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
		cli.BoolFlag{
			Name:   insecure,
			Usage:  "enable insecure interface",
			EnvVar: "RUDDER_INSECURE",
		},
	}

	app.Action = startRudder
	app.Run(os.Args)
}

func startRudder(ctx *cli.Context) error {
	// unified logging format
	isDebug := ctx.Bool(debugFlag)
	initializeLogger(isDebug)
	log.Info("Starting Rudder...")

	// main container
	container := restful.NewContainer()

	// add debug, cors, and options filter
	createBasicFilters(container, isDebug)

	// add auth filter
	if !ctx.Bool(insecure) {
		username := ctx.String(basicAuthUsernameFlag)
		password := ctx.String(basicAuthPasswordFlag)
		oidcIssuerURL := ctx.String(oidcIssuerURLFlag)
		clientID := ctx.String(clientIDFlag)
		clientSecret := ctx.String(clientSecretFlag)
		secretIsBase64Encoded := ctx.Bool(clientSecretBase64EncodedFlag)
		createAuthFilter(container, username, password, oidcIssuerURL, clientID, clientSecret, secretIsBase64Encoded)
	}

	// add `repo` resource
	repoFile := ctx.String(helmRepoFileFlag)
	cacheDir := ctx.String(helmCacheDirFlag)
	cacheLifetime := ctx.Duration(helmRepoCacheLifetimeFlag)
	repoController := createRepoController(repoFile, cacheDir, cacheLifetime)
	registerRepoResource(container, repoController)

	// add `release` resource
	tillerAddress := ctx.String(tillerAddressFlag)
	registerReleaseResource(container, repoController, tillerAddress)

	// add swagger service
	swaggerUIPath := ctx.String(swaggerUIPathFlag)
	registerSwagger(container, swaggerUIPath)

	// start server
	address := ctx.String(addressFlag)
	log.Infof("Rudder listening at: %v", address)
	return http.ListenAndServe(address, container)
}

func initializeLogger(isDebug bool) {
	// use full timestamp
	logFormat := &log.TextFormatter{FullTimestamp: true}
	log.SetFormatter(logFormat)
	// use same format for restful logger
	restfulLogger := log.New()
	restfulLogger.Formatter = logFormat
	restfullog.SetLogger(restfulLogger)
	// debug mode
	if isDebug {
		log.SetLevel(log.DebugLevel)
		log.Debug("DEBUG Mode enabled.")
	}
}

func createBasicFilters(container *restful.Container, isDebug bool) {
	// debug filter
	if isDebug {
		debugFilter := filter.NewDebugFilter()
		container.Filter(debugFilter.Debug)
		log.Info("Debug filter added.")
	}
	// cors
	cors := restful.CrossOriginResourceSharing{
		AllowedMethods: corsAllowedMethods,
		AllowedHeaders: corsAllowedHeaders,
		Container:      container,
	}
	container.Filter(cors.Filter)
	log.Info("CORS filter added.")
	// options
	container.Filter(container.OPTIONSFilter)
	log.Info("OPTIONS filter added.")
}

func createAuthFilter(container *restful.Container, username, password, oidcIssuerURL, clientID, clientSecret string, isBase64Encoded bool) {
	// supported auth
	var supportedAuth []auth.Auth
	// enable basic auth of username and password are defined
	if username != "" && password != "" {
		supportedAuth = append(supportedAuth, auth.NewBasicAuth(username, password))
	}
	// enable oidc auth if oidc issuer url or client secret is defined
	if oidcIssuerURL != "" || clientSecret != "" {
		oidcAuth := auth.NewOIDCAuth(oidcIssuerURL, clientID, clientSecret, isBase64Encoded)
		supportedAuth = append(supportedAuth, oidcAuth)
	}
	// don't include swagger to exceptions
	exceptions := []string{
		"/apidocs.json",
		"/swagger",
	}

	authFilter := auth.NewAuthFilter(supportedAuth, exceptions)
	container.Filter(authFilter.Filter)
	log.Info("Auth filter added")
}

func createRepoController(repoFileURL, cacheDir string, cacheLife time.Duration) *controller.RepoController {
	repoFileYAML, err := ioutil.ReadFile(repoFileURL)
	if err != nil {
		log.Fatalf("unable to read repo file at %s", repoFileURL)
	}
	repoFileJSON, err := yaml.YAMLToJSON(repoFileYAML)
	if err != nil {
		log.Fatal("unable to convert yaml to json")
	}
	var repoFile repo.RepoFile
	if err := json.Unmarshal(repoFileJSON, &repoFile); err != nil {
		log.Fatal("unable to parse repoFile")
	}
	repoController := controller.NewRepoController(repoFile.Repositories, cacheDir, cacheLife)
	return repoController
}

func registerRepoResource(container *restful.Container, repoController *controller.RepoController) {
	repoResource := resource.NewRepoResource(repoController)
	repoResource.Register(container)
	log.Info("repo resource registered.")
}

func registerReleaseResource(container *restful.Container, repoController *controller.RepoController, tillerAddress string) {
	tillerClient := client.NewTillerClient(tillerAddress)
	releaseController := controller.NewReleaseController(tillerClient, repoController)
	releaseResource := resource.NewReleaseResource(releaseController)
	releaseResource.Register(container)
	log.Info("release resource registered.")
}

func registerSwagger(container *restful.Container, swaggerUIPath string) {
	swaggerConfig := swagger.Config{
		WebServices: container.RegisteredWebServices(),
		ApiPath:     swaggerAPIPath,
		ApiVersion:  version,
		Info: swagger.Info{
			Title:       "Rudder",
			Description: "RESTful proxy for the Tiller service",
		},
		SwaggerPath:     swaggerPath,
		SwaggerFilePath: swaggerUIPath,
	}
	swagger.RegisterSwaggerService(swaggerConfig, container)
	log.Info("Swagger enabled.")
}
