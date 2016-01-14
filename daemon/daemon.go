package daemon

import (
	"encoding/json"
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
	InitiatorIFace string //iface to use of iSCSI initiator
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

func Start(cfgFile string, debug bool) {
	if debug == true {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	cfg, _ := processConfig(cfgFile)
	d := NewSolidFireDriverFromConfig(&cfg)
	h := volume.NewHandler(d)
	log.Info(h.ServeUnix("root", "solidfire"))
}
