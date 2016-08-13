package main

import (
	"flag"
	"github.com/Sirupsen/logrus"
	"github.com/cloudfoundry-incubator/candiedyaml"
	"os"
	"path"
)

var (
	argLibraryPath *string = flag.String("library", "", "Path to 'iTunes Music Library.xml'")
	argTargetPath  *string = flag.String("target", "", "Path to sync target directory")
	argVerbose     *bool   = flag.Bool("verbose", false, "Verbose output")
	argDryRun      *bool   = flag.Bool("dryrun", false, "DryRun mode")
)

type Config struct {
	Playlists []string `yaml:"playlists"`
}

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

func isValidWalkmanDevice(devicePath string) bool {
	return isFileExists(path.Join(devicePath, "capability_00.xml")) ||
		isFileExists(path.Join(devicePath, "default-capability.xml"))
}

func findTargetPath() (string, bool) {
	if argTargetPath != nil && *argTargetPath != "" {
		return *argTargetPath, true
	} else {
		candidates := listDeviceCandidates()
		switch len(candidates) {
		case 0:
			return "", false
		case 1:
			return candidates[0], true
		default:
			logrus.Warnf("Too many device candidates found!(%v): Using %s", candidates, candidates[0])
			return candidates[0], true
		}
	}
}

func main() {
	flag.Parse()
	confFp, err := os.Open("config.yaml")
	if err != nil {
		logrus.Fatalf("Config not found: %s", err)
	}
	decoder := candiedyaml.NewDecoder(confFp)
	var config Config
	err = decoder.Decode(&config)
	if err != nil {
		logrus.Fatalf("Could not parse config: %s", err)
	}
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
	logrus.Infof("Playlists: %v", config.Playlists)
	if *argDryRun {
		logrus.Infof("============ DRYRUN Mode ==============")
	}
	err = startSync(libPath, targetPath, config.Playlists)
	if err != nil {
		logrus.Fatalf("Error: %s", err)
	}
}
