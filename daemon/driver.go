package daemon

import (
	log "github.com/Sirupsen/logrus"
	"github.com/alecthomas/units"
	"os"
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

func NewSolidFireDriverFromConfig(c *sfapi.Config) SolidFireDriver {
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
			log.Fatal("Failed to initialize solidfire driver while creating tenant: ", err)
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

	defaultVolSize := int64(1)
	if c.DefaultVolSize != 0 {
		defaultVolSize = c.DefaultVolSize
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
		DefaultVolSz:   defaultVolSize,
		MountPoint:     c.MountPoint,
		InitiatorIFace: iscsiInterface,
	}
	log.Debugf("Driver initialized with the following settings:\n%+v\n", d)
	log.Info("Succesfuly initialized SolidFire Docker driver")
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
		s, _ := strconv.ParseInt(r.Options["size"], 10, 64)
		log.Info("Received size request in Create: ", s)
		vsz = int64(units.GiB) * s
	} else {
		vsz = d.DefaultVolSz * int64(units.GiB)
		log.Info("Creating with default size of: ", vsz)
	}

	// TODO(jdg): These do nothing right now, add them in round 2
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
	_, err := d.Client.CreateVolume(&req)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{}
}

func (d SolidFireDriver) Remove(r volume.Request) volume.Response {
	log.Info("Remove/Delete Volume: ", r.Name)
	v, err := d.Client.GetVolume(0, r.Name)
	if err != nil {
		log.Error("Failed to retrieve volume named ", r.Name, "during Remove operation: ", err)
		return volume.Response{Err: err.Error()}
	}
	d.Client.DetachVolume(v)
	err = d.Client.DeleteVolume(v.VolumeID)
	if err != nil {
		// FIXME(jdg): Check if it's a "DNE" error in that case we're golden
		log.Error("Error encountered during delete: ", err)
	}
	return volume.Response{}
}

func (d SolidFireDriver) Path(r volume.Request) volume.Response {
	log.Info("Retrieve path info for volume: ", r.Name)
	path := filepath.Join(d.MountPoint, r.Name)
	log.Debug("Path reported as: ", path)
	return volume.Response{Mountpoint: path}
}

func (d SolidFireDriver) Mount(r volume.Request) volume.Response {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	log.Infof("Mounting volume %s on %s\n", r.Name, "solidfire")
	v, err := d.Client.GetVolume(0, r.Name)
	if err != nil {
		log.Errorf("Failed to retrieve volume by name in mount operation: ", r.Name)
		return volume.Response{Err: err.Error()}
	}
	path, device, err := d.Client.AttachVolume(&v, d.InitiatorIFace)
	if path == "" || device == "" && err == nil {
		log.Error("Missing path or device, but err not set?")
		log.Debug("Path: ", path, ",Device: ", device)
		return volume.Response{Err: err.Error()}

	}
	if err != nil {
		log.Errorf("Failed to perform iscsi attach of volume %s: %v", r.Name, err)
		return volume.Response{Err: err.Error()}
	}
	log.Debugf("Attached volume at (path, devfile): %s, %s", path, device)
	if sfapi.GetFSType(device) == "" {
		//TODO(jdg): Enable selection of *other* fs types
		err := sfapi.FormatVolume(device, "ext4")
		if err != nil {
			log.Errorf("Failed to format device: ", device)
			return volume.Response{Err: err.Error()}
		}
	}
	if sfapi.Mount(device, d.MountPoint+"/"+r.Name) != nil {
		log.Error("Failed to mount volume: ", r.Name)
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{Mountpoint: d.MountPoint + "/" + r.Name}
}

func (d SolidFireDriver) Unmount(r volume.Request) volume.Response {
	log.Info("Unmounting volume: ", r.Name)
	sfapi.Umount(filepath.Join(d.MountPoint, r.Name))
	v, err := d.Client.GetVolume(0, r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	d.Client.DetachVolume(v)
	return volume.Response{}
}
