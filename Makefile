APP=rudder
VERSION=latest
LDFLAGS=-ldflags "-X github.com/AcalephStorage/rudder/cmd.version=${VERSION}"

all: deps build

clean:
	@echo "--> cleaning..."
	@rm -rf build
	@rm -rf vendor
	@go clean ./...

prereq:
	@mkdir -p build/{bin,tar}

deps:
	@glide install

build: prereq
	@echo 'building...'
	@go fmt ./...
	@go build -o build/bin/${APP} ${LDFLAGS} ./cmd

package: build


# docker: