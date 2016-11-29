package util

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/ghodss/yaml"
)

// EncodeMD5Hex encodes in to md5 then hex
func EncodeMD5Hex(in string) string {
	hasher := md5.New()
	hasher.Write([]byte(in))
	return hex.EncodeToString(hasher.Sum(nil))

}

// YAMLtoJSON converts the YAML in to JSON out
func YAMLtoJSON(in []byte, out interface{}) (err error) {
	jsn, err := yaml.YAMLToJSON(in)
	if err != nil {
		return
	}
	err = json.Unmarshal(jsn, out)
	if err != nil {
	}
	return
}
