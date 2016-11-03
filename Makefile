APP=rudder
VERSION=latest
LDFLAGS=-ldflags "-X github.com/AcalephStorage/rudder/cmd.version=${VERSION}"

OS=`go env GOOS`
ARCH=`go env GOARCH`

all: deps build

clean:
	@echo "--> cleaning..."
	@rm -rf build
	@rm -rf vendor
	@go clean ./...

prereq:
	@mkdir -p build/{bin,tar}
	@go get -u github.com/Masterminds/glide

deps: prereq
	@glide install

build: prereq
	@echo '--> building...'
	@go fmt ./...
	@GOOS=${OS} GOARCH=${ARCH} go build -o build/bin/${APP} ${LDFLAGS} ./cmd

package: build
	@echo '--> packaging...'
	@docker build -t quay.io/acaleph/rudder:${VERSION} .

deploy: package
	@echo '--> deploying...'
	@docker push quay.io/acaleph/rudder:${VERSION}
