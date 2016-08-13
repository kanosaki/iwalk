package main

import (
	"flag"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/cloudfoundry-incubator/candiedyaml"
	"os"
	"path"
)

var (
	argLibraryPath     *string = flag.String("library", "", "Path to 'iTunes Music Library.xml'")
	argTargetPath      *string = flag.String("target", "", "Path to sync target directory")
	argVerbose         *bool   = flag.Bool("v", false, "Verbose output (Info output)")
	argDebug           *bool   = flag.Bool("vv", false, "More verbose output(Debug output)")
	argDryRun          *bool   = flag.Bool("dryrun", false, "DryRun mode")
	argPrintLibSummary *bool   = flag.Bool("print_library", false, "Print iTunes Library summary and exit with do nothing.")
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

func printLibrarySummary(libPath string) {
	l, err := LoadLibrary(libPath)
	if err != nil {
		logrus.Fatalf("Failed to load library", err)
	}
	fmt.Println("-------- Playlists ------------")
	for _, playlist := range l.Playlists {
		fmt.Printf("%s: %d tracks\n", playlist.Name, len(playlist.PlaylistItems))
	}
}

func main() {
	flag.Parse()
	logrus.SetLevel(logrus.WarnLevel)
	if *argVerbose {
		logrus.SetLevel(logrus.InfoLevel)
	}
	if *argDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	libPath, ok := findLibraryPath()
	if !ok {
		logrus.Fatalf("Library.xml not found!")
	} else {
		logrus.Infof("Library: %s", libPath)
	}
	if *argPrintLibSummary {
		printLibrarySummary(libPath)
		return
	}
	confFp, err := os.Open(os.ExpandEnv("$HOME/.config/iwalk.yaml"))
	if err != nil {
		logrus.Fatalf("Config not found: %s", err)
	}
	decoder := candiedyaml.NewDecoder(confFp)
	var config Config
	err = decoder.Decode(&config)
	if err != nil {
		logrus.Fatalf("Could not parse config: %s", err)
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
