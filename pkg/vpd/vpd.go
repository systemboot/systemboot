package vpd

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

var (
	VpdDir = "/sys/firmware/vpd"
)

func getBaseDir(readOnly bool) string {
	var baseDir string
	if readOnly {
		baseDir = path.Join(VpdDir, "ro")
	} else {
		baseDir = path.Join(VpdDir, "rw")
	}
	return baseDir
}

func Get(key string, readOnly bool) ([]byte, error) {
	buf, err := ioutil.ReadFile(path.Join(getBaseDir(readOnly), key))
	if err != nil {
		return []byte{}, err
	}
	return buf, nil
}

func Set(key string, value []byte, readOnly bool) error {
	// NOTE this is not implemented yet in the kernel interface, and will always
	// return a permission denied error
	return ioutil.WriteFile(path.Join(getBaseDir(readOnly), key), value, 0644)
}

func GetAll(readOnly bool) (map[string][]byte, error) {
	vpdMap := make(map[string][]byte, 0)
	baseDir := getBaseDir(readOnly)
	err := filepath.Walk(baseDir, func(fpath string, info os.FileInfo, err error) error {
		key := path.Base(fpath)
		if key == "." || key == "/" || fpath == baseDir {
			// empty or all slashes?
			return nil
		}
		value, err := Get(key, readOnly)
		if err != nil {
			return err
		}
		vpdMap[key] = value
		return nil
	})
	return vpdMap, err
}
