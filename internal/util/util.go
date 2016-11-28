package util

import (
	"bytes"
	"io"
	"time"

	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"crypto/md5"
	"encoding/hex"
	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"github.com/ghodss/yaml"
	"strconv"
)

var ErrFailToReadResponse = restful.NewError(http.StatusBadRequest, "unable to read request body")
var ErrFailToWriteResponse = restful.NewError(http.StatusInternalServerError, "unable to write response")

func ErrorResponse(res *restful.Response, err restful.ServiceError) {
	log.WithError(err).Error(err.Message)
	if err := res.WriteServiceError(err.Code, err); err != nil {
		log.WithError(err).Error("unable to write error")
	}
}

func IsOutdated(timestamp time.Time, lifetime time.Duration) bool {
	elapsed := time.Now().Sub(timestamp)
	return elapsed >= lifetime
}

func isExpired(timestamp time.Time) bool {
	return time.Now().After(timestamp)
}

func HttpGET(url string) (out []byte, err error) {
	res, err := http.Get(url)
	if err == nil {
		defer res.Body.Close()
		out, err = ioutil.ReadAll(res.Body)
	}
	return
}

func EncodeMD5Hex(in string) string {
	hasher := md5.New()
	hasher.Write([]byte(in))
	return hex.EncodeToString(hasher.Sum(nil))

}

func ReadFile(file string) (out []byte, err error) {
	return ioutil.ReadFile(file)
}

func WriteFile(file string, data []byte) error {
	return ioutil.WriteFile(file, data, 0644)
}

func YAMLtoJSON(in []byte, out interface{}) (err error) {
	jsn, err := yaml.YAMLToJSON(in)
	if err != nil {
		log.Debugf("unable to parse from YAML to JSON")
		return
	}
	err = json.Unmarshal(jsn, out)
	if err != nil {
		log.Debugf("unable to unmarshal JSON")
	}
	return
}

func TarballToMap(in []byte) (out map[string][]byte, err error) {
	byteReader := bytes.NewReader(in)
	gzipReader, err := gzip.NewReader(byteReader)
	defer gzipReader.Close()
	if err != nil {
		return
	}
	tarReader := tar.NewReader(gzipReader)
	out = make(map[string][]byte)
	for {
		header, tarErr := tarReader.Next()
		if tarErr == io.EOF {
			// eof
			break
		}
		if tarErr != nil {
			// something went wrong
			err = tarErr
			return
		}
		// only regular files
		info := header.FileInfo()
		if info.IsDir() {
			continue
		}
		data, readErr := ioutil.ReadAll(tarReader)
		if readErr != nil {
			err = readErr
			return
		}
		out[header.Name] = data

	}
	return
}

func ToInt32(in string) (out int32) {
	val, _ := strconv.ParseInt(in, 10, 32)
	return int32(val)
}
