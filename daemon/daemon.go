package daemon

import (
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/j-griffith/solidfire-docker-driver/sfapi"
	"path/filepath"
)

var (
	defaultDir = filepath.Join(volume.DefaultDockerRootDirectory, "solidfire")
)

func Start(cfgFile string, debug bool) {
	if debug == true {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	cfg, _ := sfapi.ProcessConfig(cfgFile)
	d := NewSolidFireDriverFromConfig(&cfg)
	h := volume.NewHandler(d)
	log.Info(h.ServeUnix("root", "solidfire"))
}
