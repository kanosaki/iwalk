package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"
	"syscall"
)

const VOLUMES = "/Volumes"

const (
	R_OK uint32 = 4
	W_OK uint32 = 2
	X_OK uint32 = 1
	F_OK uint32 = 0
)

var VOLUMES_IGNORES = []string{"Macintosh HD", "MobileBackups", "Time Machine"}

func isWindows() bool {
	return false
}

func defaultLibraryPath() string {
	return fmt.Sprintf("/Users/%v/Music/iTunes/iTunes Music Library.xml", os.Getenv("USER"))
}

func normalizeLocation(path string) string {
	url, err := url.Parse(path)
	if err != nil {
		logrus.Fatalf("Invlaid Location URL: %s", path)
		return ""
	} else {
		return url.Path
	}
}

var hfsPlusReplacer = strings.NewReplacer(
	"/", "_", // UNIX rule
	"\x00", "_", // HFS+ rule
)

func escapeFilename(name string) string {
	return hfsPlusReplacer.Replace(name)
}

func listDeviceCandidates() []string {
	fInfos, err := ioutil.ReadDir(VOLUMES)
	if err != nil {
		return []string{}
	}
	ret := make([]string, 0, len(fInfos))
	for _, info := range fInfos {
		devicePath := path.Join(VOLUMES, info.Name())
		if !info.IsDir() {
			goto SKIP_CANDIDATE
		}
		for _, ignorePattern := range VOLUMES_IGNORES {
			if strings.Contains(info.Name(), ignorePattern) {
				goto SKIP_CANDIDATE
			}
		}
		if isValidWalkmanDevice(devicePath) {
			ret = append(ret, path.Join(devicePath, "MUSIC"))
		}
	SKIP_CANDIDATE:
	}
	return ret
}

func isReadable(path string) bool {
	err := syscall.Access(path, R_OK)
	return err == nil
}

func isWritable(path string) bool {
	err := syscall.Access(path, W_OK)
	return err == nil
}
