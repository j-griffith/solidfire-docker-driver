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
	TenantName     string
	TenantID       int64
	DefaultVolSz   int64
	VagID          int64
	MountPoint     string
	InitiatorIFace string
	Client         *sfapi.Client
	Mutex          *sync.Mutex
}

func NewSolidFireDriverFromConfig(c *Config) SolidFireDriver {
	var tenantID int64

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
		if err != nil {
			log.Fatalf("Failed to initialize solidfire driver creating tenant: ", err)
		}
	} else {
		tenantID = account.AccountID
	}

	baseMountPoint := "/var/lib/solidfire/mount"
	if c.MountPoint != "" {
		baseMountPoint = c.MountPoint
	}

	iscsiInterface := "default"

	if c.InitiatorIFace != "" {
		iscsiInterface = c.InitiatorIFace
	}

	_, err = os.Lstat(baseMountPoint)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(baseMountPoint, 0755); err != nil {
			log.Fatal("Failed to create Mount directory during driver init: %v", err)
		}
	}

	d := SolidFireDriver{
		TenantName:     c.TenantName,
		TenantID:       tenantID,
		Client:         client,
		Mutex:          &sync.Mutex{},
		DefaultVolSz:   1,
		MountPoint:     c.MountPoint,
		InitiatorIFace: iscsiInterface,
	}
	log.Debugf("Succesfuly initialized SolidFire Docker driver")
	return d
}

func (d SolidFireDriver) Create(r volume.Request) volume.Response {
	// TODO(jdg):  Add a check of options her that looks for an FS-Type
	// specifier.  What we'll do is add it to the attributes of the volume on
	// create, then during mount, we can check that, and format it to the
	// requested type.  Just be sure that when we do that we pop the attribute
	// off so we don't "reformat" :)
	// For now we just do ext4 only
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
	// TODO(jdg):  Make sure it's actually mounted :)
	return volume.Response{Mountpoint: filepath.Join(d.MountPoint, r.Name)}
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
	//TODO(jdg): Ensure we're not already attached or mounted
	path, device, err := d.Client.AttachVolume(&v, d.InitiatorIFace)
	if path == "" || device == "" && err == nil {
		log.Error("Missing path or device, but err not set?")
		log.Debug("Path: ", path, ",Device: ", device)
		return volume.Response{Err: err.Error()}

	}
	if err != nil {
		log.Errorf("Failed to perform iscsi attach of volume %s: %s", err)
		return volume.Response{Err: err.Error()}
	}
	log.Debugf("Attached volume at (path, devfile): %s, %s", path, device)
	if sfapi.GetFSType(device) == "" {
		if sfapi.FormatVolume(device, "ext4") != nil {
			log.Errorf("Failed to format device: ", device)
			return volume.Response{Err: err.Error()}
		}
	}
	// Mount it to the local mountpoint
	if sfapi.Mount(device, d.MountPoint+"/"+r.Name) != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{Mountpoint: d.MountPoint + "/" + r.Name}
}

func (d SolidFireDriver) Unmount(r volume.Request) volume.Response {
	sfapi.Umount(filepath.Join(d.MountPoint, r.Name))
	d.Client.DetachVolume(0, r.Name)
	return volume.Response{}
}
