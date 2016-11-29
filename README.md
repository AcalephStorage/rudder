rudder
------

RESTful API for Helm Repositories and the Tiller service.

## Requirements

 - Tiller v2.0.0

## Installation

### Binary

Binaries can be downloaded [here](https://github.com/AcalephStorage/rudder/releases).

### Docker

Docker images are also available:

```
$ docker pull quay.io/acaleph/rudder
```

## Running

```
$ rudder {{flags}}
```

or

```
$ docker run quay.io/acaleph/rudder
```

## Configuration

Configuration can be provided via cli flags or through environment variables:

| Configuration         | Flag                           | Environment Variable            | Default                              |
|-----------------------|--------------------------------|---------------------------------|--------------------------------------|
| Rudder address        | --address                      | RUDDER_ADDRESS                  | 0.0.0.0:5000                         |
| Tiller address        | --tiller-address               | RUDDER_TILLER_ADDRESS           | localhost:44134                      |
| Repo File             | --helm-repo-file               | RUDDER_HELM_REPO_FILE           | ~/.helm/repository/repositories.yaml |
| Cache Directory       | --helm-cache-dir               | RUDDER_HELM_CACHE_DIR           | /opt/rudder/cache                    |
| Cache Lifetime        | --helm-repo-cache-lifetime     | RUDDER_HELM_REPO_CACHE_LIFETIME | 10m                                  |
| Swagger UI Path       | --swagger-ui-path              | RUDDER_SWAGGER_UI_PATH          | /opt/rudder/swagger                  |
| Basic Auth Username   | --basic-auth-username          | RUDDER_BASIC_AUTH_USERNAME      |                                      |
| Basic Auth Password   | --basic-auth-password          | RUDDER_BASIC_AUTH_PASSWORD      |                                      |
| OIDC Issuer URL       | --oidc-issuer-url              | RUDDER_OIDC_ISSUER_URL          |                                      |
| Client ID             | --client-id                    | RUDDER_CLIENT_ID                |                                      |
| Client Secret         | --client-secret                | RUDDER_CLIENT_SECRET            |                                      |
| Client Secret Encoded | --client-secret-base64-encoded | RUDDER_CLIENT_BASE64_ENCODED    |                                      |
| Debug Mode            | --debug                        |                                 |                                      |

## API

API docs is provided via swagger. This is available at: `http://{rudder-url}/swagger`.

Using the docker image already has this enabled by default. When using the binary, copy the [swagger files](https://github.com/AcalephStorage/rudder/tree/develop/third-party/swagger) to `/opt/rudder/swagger` or a different directory and set `--swagger-ui-path`.

Currently there are read-only Helm Repository endpoints for fetching charts from repositories and Basic Release endpoints (tiller), `install` and `uninstall`. The rest is still WIP.

## Notes

### Helm Repositories

At the moment, repositories are provided via a repo file. The format should be the same as what helm uses (`~/.helm/repository/repositories.yaml`). This may change in the future when a repo manager is implemented.

### Charts cache

Charts are downloaded from the helm repository and are cached at the location defined by `--helm-cache-dir` (default: ./opt/rudder/cache). This directory should exist and be writable.

### Authentication

Authentication can be enabled by providing authentication details.

#### Basic Auth

Providing `--basic-auth-username` and `--basic-auth-password` will enable Basic Authentication.

#### OIDC

Providing `--oidc-issuer-url` or `--client-secret` will enable OIDC.

## TODO

This is still WIP. Some immediate TODOs are:

 - [ ] implement a repo manager
 - [ ] implement missing tiller functions
