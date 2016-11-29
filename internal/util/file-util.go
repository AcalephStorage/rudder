package util

import (
	"bytes"
	"io"

	"archive/tar"
	"compress/gzip"
	"io/ioutil"
)

// ReadFile reads the file from the given filePath
func ReadFile(filePath string) (out []byte, err error) {
	return ioutil.ReadFile(filePath)
}

// WriteFile writes the data to the given filePath
func WriteFile(filePath string, data []byte) error {
	return ioutil.WriteFile(filePath, data, 0644)
}

// TarballToMap converts a tarball to map[string][]byte
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
