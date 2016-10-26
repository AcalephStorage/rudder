package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	restfullog "github.com/emicklei/go-restful/log"
	"github.com/emicklei/go-restful/swagger"
	"github.com/urfave/cli"

	"github.com/AcalephStorage/rudder/client"
	"github.com/AcalephStorage/rudder/filter"
	"github.com/AcalephStorage/rudder/resource"
	"net/http"
)

const (
	appName = "rudder"

	addressFlag       = "address"
	tillerAddressFlag = "tiller-address"
	authUsernameFlag  = "auth-username"
	authPasswordFlag  = "auth-password"
	swaggerUIPathFlag = "swagger-ui-path"
	debugFlag         = "debug"
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
		},
		cli.StringFlag{
			Name:   authPasswordFlag,
			Usage:  "basic auth password",
			EnvVar: "RUDDER_AUTH_PASSWORD",
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
	restfullog.SetLogger(restfulLogger)

	log.Info("Starting Rudder...")

	// flags
	address := ctx.String(addressFlag)
	tillerAddress := ctx.String(tillerAddressFlag)
	authUsername := ctx.String(authUsernameFlag)
	authPassword := ctx.String(authPasswordFlag)
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

	// tiller client
	tillerClient := client.NewTillerClient(tillerAddress)
	// release resource
	releaseResource := resource.NewReleaseResource(tillerClient)
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
