package main

import (
	"flag"
	"github.com/Sirupsen/logrus"
	"os"
)

var (
	argLibraryPath *string = flag.String("-library", "", "Path to 'iTunes Music Library.xml'")
	argTargetPath  *string = flag.String("-target", "", "Path to sync target directory")
	argVerbose     *string = flag.String("-verbose", "", "Verbose output")
)

func findLibraryPath() (string, bool) {
	logrus.Debugf("Finding library...")
	ret := *argLibraryPath
	logrus.Debugf("Checking argument: %s", *argLibraryPath)
	if ret == "" {
		defaultLibPath := defaultLibraryPath()
		logrus.Debugf("Checking default library: %s", defaultLibPath)
		_, err := os.Stat(defaultLibPath)
		if err == nil || !os.IsNotExist(err) {
			return defaultLibPath, true
		} else {
			return "", false
		}
	} else {
		return ret, true
	}
}

func findTargetPath() (string, bool) {

}

func main() {
	flag.Parse()
	if *argVerbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
	libPath, ok := findLibraryPath()
	if !ok {
		logrus.Fatalf("Library.xml not found!")
	} else {
		logrus.Infof("Library: %s", libPath)
	}

	targetPath, ok := findTargetPath()
	if !ok {
		logrus.Fatalf("SyncTarget not found!")
	} else {
		logrus.Infof("Target: %s", targetPath)
	}
	startSync(libPath, targetPath)
}
