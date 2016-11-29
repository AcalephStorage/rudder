package util

import (
	"io/ioutil"
	"net/http"
)

// HTTPGet is a convenience method for quicking GETting an HTTP resource to a []byte
func HTTPGet(url string) (out []byte, err error) {
	res, err := http.Get(url)
	if err == nil {
		defer res.Body.Close()
		out, err = ioutil.ReadAll(res.Body)
	}
	return
}
