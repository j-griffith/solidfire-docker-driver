package main

import (
	"encoding/json"
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"io/ioutil"
	"path/filepath"
)

var (
	defaultDir = filepath.Join(volume.DefaultDockerRootDirectory, "solidfire")
)

type Config struct {
	TenantName     string
	EndPoint       string
	DefaultVolSize int64 //Default volume size in GiB
	MountPoint     string
	SVIP           string
	//Types       []map[string]QoS
}

func processConfig(fname string) (Config, error) {
	content, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Fatal("Error processing config file: ", err)
	}
	var conf Config
	err = json.Unmarshal(content, &conf)
	if err != nil {
		log.Fatal("Error parsing config file: ", err)
	}
	return conf, nil
}

func main() {
	cfgFile := flag.String("config", "/var/lib/solidfire/solidfire.json", "Configuration file for SolidFire Docker Daemon.")
	debug := flag.Bool("debug", false, "Enable debug logging.")
	log.SetLevel(log.DebugLevel)
	flag.Parse()
	if *debug == true {
		log.SetLevel(log.DebugLevel)
	}

	cfg, _ := processConfig(*cfgFile)
	log.Info(cfg.EndPoint)
	d := NewSolidFireDriverFromConfig(&cfg)
	h := volume.NewHandler(d)
	log.Info(h.ServeUnix("root", "solidfire"))
}
