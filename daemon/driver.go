package main

import (
	//	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/alecthomas/units"
	"os"
	//	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/j-griffith/solidfire-docker-driver/sfapi"
)

type SolidFireDriver struct {
	TenantName   string
	TenantID     int64
	DefaultVolSz int64
	VagID        int64
	MountPoint   string
	Client       *sfapi.Client
	Mutex        *sync.Mutex
}

type Configuration struct {
	Endpoint          string
	TenantName        string
	DefaultMountPoint string
	DefaultVolSize    int64
	QoSTypes          struct {
	}
}

func NewSolidFireDriverFromConfig(c *Config) SolidFireDriver {
	var tenantID int64
	var listVagReq sfapi.ListVolumeAccessGroupsRequest
	var createVagReq sfapi.CreateVolumeAccessGroupRequest
	var vagID int64
	log.SetLevel(log.DebugLevel)

	client, _ := sfapi.NewWithArgs(c.EndPoint, c.SVIP, c.TenantName, c.DefaultVolSize)
	req := sfapi.GetAccountByNameRequest{
		Name: c.TenantName,
	}

	account, err := client.GetAccountByName(&req)
	if err != nil {
		req := sfapi.AddAccountRequest{
			Username: c.TenantName,
		}
		tenantID, err = client.AddAccount(&req)
	} else {
		tenantID = account.AccountID
	}

	vags, err := client.ListVolumeAccessGroups(&listVagReq)
	if err != nil {
	}
	for _, v := range vags {
		if v.Name == c.TenantName {
			vagID = v.VAGID
		}
	}
	if vagID == 0 {
		createVagReq.Name = c.TenantName
		vagID, _ = client.CreateVolumeAccessGroup(&createVagReq)
		client.CreateVolumeAccessGroup(&createVagReq)
	}

	baseMountPoint := "/var/lib/solidfire/mount"
	if c.MountPoint != "" {
		baseMountPoint = c.MountPoint
	}

	_, err = os.Lstat(baseMountPoint)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(baseMountPoint, 0755); err != nil {
			log.Fatal("Failed to create Mount directory during driver init: %v", err)
		}
	}

	d := SolidFireDriver{
		TenantName:   c.TenantName,
		TenantID:     tenantID,
		Client:       client,
		VagID:        vagID,
		Mutex:        &sync.Mutex{},
		DefaultVolSz: 1,
		MountPoint:   c.MountPoint,
	}
	return d
}

func (d SolidFireDriver) Create(r volume.Request) volume.Response {
	log.Infof("Create volume %s on %s\n", r.Name, "solidfire")
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	var req sfapi.CreateVolumeRequest
	var qos sfapi.QoS
	var vsz int64

	if r.Options["size"] != "" {
		log.Debugf("Requested size is: ", r.Options["size"])
		s, _ := strconv.ParseInt(r.Options["size"], 10, 64)
		vsz = int64(units.GiB) * s
	} else {
		vsz = d.DefaultVolSz * int64(units.GiB)
		log.Debugf("Using default size of: ", vsz)
	}

	if r.Options["qos"] != "" {
		iops := strings.Split(r.Options["qos"], ",")
		qos.MinIOPS, _ = strconv.ParseInt(iops[0], 10, 64)
		qos.MaxIOPS, _ = strconv.ParseInt(iops[1], 10, 64)
		qos.BurstIOPS, _ = strconv.ParseInt(iops[2], 10, 64)
		req.Qos = qos
	}

	req.TotalSize = vsz
	req.AccountID = d.TenantID
	req.Name = r.Name
	log.Debugf("Create volume with request: ", &req)
	_, err := d.Client.CreateVolume(&req)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{}
}

func (d SolidFireDriver) Remove(r volume.Request) volume.Response {
	log.Printf("Remove volume %s on %s\n", r.Name, "somewhere")
	return volume.Response{Mountpoint: "foo"}
}

func (d SolidFireDriver) Path(r volume.Request) volume.Response {
	log.Printf("Fetching Path for volume %s on %s\n", r.Name, "solidfire")
	return volume.Response{Mountpoint: filepath.Join("/var/lib/mount", r.Name)}
}

func (d SolidFireDriver) Mount(r volume.Request) volume.Response {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	log.Printf("Mounting volume %s on %s\n", r.Name, "solidfire")
	v, err := d.Client.GetVolume(0, r.Name)
	if err != nil {
		log.Errorf("Failed to retrieve volume by name in mount operation: ", r.Name)
		return volume.Response{Err: err.Error()}
	}
	path, device, err := d.Client.AttachVolume(&v)
	if err != nil {
		log.Errorf("Failed to perform iscsi attach of volume %s: %s", err)
		return volume.Response{Err: err.Error()}
	}
	log.Debugf("Attached volume at (path, devfile): %s, %s", path, device)
	// Mount it to the local mountpoint
	/*
		s, ok := d.volumes[m]
		if ok && s.connections > 0 {
			s.connections++
			return volume.Response{Mountpoint: m}
		}

		fi, err := os.Lstat(m)

		if os.IsNotExist(err) {
			if err := os.MkdirAll(m, 0755); err != nil {
				return volume.Response{Err: err.Error()}
			}
		} else if err != nil {
			return volume.Response{Err: err.Error()}
		}

		if fi != nil && !fi.IsDir() {
			return volume.Response{Err: fmt.Sprintf("%v already exist and it's not a directory", m)}
		}

		if err := d.mountVolume(r.Name, m); err != nil {
			return volume.Response{Err: err.Error()}
		}

		d.volumes[m] = &volume_name{name: r.Name, connections: 1}
	*/
	return volume.Response{Mountpoint: "foo"}
}

func (d SolidFireDriver) Unmount(r volume.Request) volume.Response {
	log.Printf("Unmount volume %s on %s\n", r.Name, "somewhere")
	return volume.Response{Mountpoint: "foo"}
}
