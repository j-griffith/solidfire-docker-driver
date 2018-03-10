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
	"github.com/j-griffith/sfgo/pkg/provider"
	sfc "github.com/j-griffith/sfgo/pkg/client"
)

type SolidFireDriver struct {
	TenantID       int64
	DefaultVolSz   int64
	VagID          int64
	MountPoint     string
	InitiatorIFace string
	Client         *sfapi.Client
	Mutex          *sync.Mutex
}

func verifyConfiguration(cfg *sfapi.Config) {
	// We want to verify we have everything we need to run the Docker driver
	if cfg.TenantName == "" {
		log.Fatal("TenantName required in SolidFire Docker config")
	}
	if cfg.EndPoint == "" {
		log.Fatal("EndPoint required in SolidFire Docker config")
	}
	if cfg.DefaultVolSz == 0 {
		log.Fatal("DefaultVolSz required in SolidFire Docker config")
	}
	if cfg.SVIP == "" {
		log.Fatal("SVIP required in SolidFire Docker config")
	}
}
func New(cfgFile string) SolidFireDriver {
	var tenantID int64
	// sfc.InitClient("/var/lib/config")
	// sfc.GetSolidFireClient()
	client, _ := sfapi.NewFromConfig(cfgFile)

	req := sfapi.GetAccountByNameRequest{
		Name: client.DefaultTenantName,
	}
	account, err := client.GetAccountByName(&req)
	if err != nil {
		req := sfapi.AddAccountRequest{
			Username: client.DefaultTenantName,
		}
		actID, err := client.AddAccount(&req)
		if err != nil {
			log.Fatalf("Failed init, unable to create Tenant (%s): %+v", client.DefaultTenantName, err)
		}
		tenantID = actID
		log.Debug("Set tenantID: ", tenantID)
	} else {
		tenantID = account.AccountID
		log.Debug("Set tenantID: ", tenantID)
	}
	baseMountPoint := "/var/lib/solidfire/mount"
	if client.Config.MountPoint != "" {
		baseMountPoint = client.Config.MountPoint
	}

	iscsiInterface := "default"
	if client.Config.InitiatorIFace != "" {
		iscsiInterface = client.Config.InitiatorIFace
	}

	_, err = os.Lstat(baseMountPoint)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(baseMountPoint, 0755); err != nil {
			log.Fatal("Failed to create Mount directory during driver init: %v", err)
		}
	}

	d := SolidFireDriver{
		TenantID:       tenantID,
		Client:         client,
		Mutex:          &sync.Mutex{},
		DefaultVolSz:   client.DefaultVolSize,
		MountPoint:     client.Config.MountPoint,
		InitiatorIFace: iscsiInterface,
	}
	return d
}

func NewSolidFireDriverFromConfig(c *sfapi.Config) SolidFireDriver {
	var tenantID int64

	client, _ := sfapi.NewFromConfig("")
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

	if c.Types != nil {
		client.VolumeTypes = c.Types
	}

	defaultVolSize := int64(1)
	if c.DefaultVolSz != 0 {
		defaultVolSize = c.DefaultVolSz
	}

	_, err = os.Lstat(baseMountPoint)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(baseMountPoint, 0755); err != nil {
			log.Fatal("Failed to create Mount directory during driver init: %v", err)
		}
	}

	d := SolidFireDriver{
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

func formatOpts(r *volume.CreateRequest) {
	// NOTE(jdg): For now we just want to minimize issues like case usage for
	// the two basic opts most used (size and type).  Going forward we can add
	// all sorts of things here based on what we decide to add as valid opts
	// during create and even other calls
	for k, v := range r.Options {
		if strings.EqualFold(k, "size") {
			r.Options["size"] = v
		} else if strings.EqualFold(k, "type") {
			r.Options["type"] = v
		} else if strings.EqualFold(k, "qos") {
			r.Options["qos"] = v
		}
	}
}

func (d SolidFireDriver) Create(r *volume.CreateRequest) error {
	log.Infof("Create volume %s on %s\n", r.Name, "solidfire")
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	var req sfapi.CreateVolumeRequest
	var qos sfapi.QoS
	var vsz int64

	log.Debugf("GetVolumeByName: %s, %d", r.Name, d.TenantID)
	log.Debugf("Options passed in to create: %+v", r.Options)
	v, err := d.Client.GetVolumeByName(r.Name, d.TenantID)
	if err == nil && v.VolumeID != 0 {
		log.Infof("Found existing Volume by Name: %s", r.Name)
		return err
	}
	formatOpts(r)
	log.Debugf("Options after conversion: %+v", r.Options)
	if r.Options["size"] != "" {
		s, _ := strconv.ParseInt(r.Options["size"], 10, 64)
		log.Info("Received size request in Create: ", s)
		vsz = int64(units.GiB) * s
	} else {
		// NOTE(jdg): We need to cleanup the conversions and such when we read
		// in from the config file, it's sort of ugly.  BUT, just remember that
		// when we pull the value from d.DefaultVolSz it's already been
		// multiplied
		vsz = d.DefaultVolSz
		log.Info("Creating with default size of: ", vsz)
	}

	if r.Options["qos"] != "" {
		iops := strings.Split(r.Options["qos"], ",")
		qos.MinIOPS, _ = strconv.ParseInt(iops[0], 10, 64)
		qos.MaxIOPS, _ = strconv.ParseInt(iops[1], 10, 64)
		qos.BurstIOPS, _ = strconv.ParseInt(iops[2], 10, 64)
		req.Qos = qos
		log.Infof("Received qos r.Options in Create: %+v", req.Qos)
	}

	if r.Options["type"] != "" {
		for _, t := range *d.Client.VolumeTypes {
			if strings.EqualFold(t.Type, r.Options["type"]) {
				req.Qos = t.QOS
				log.Infof("Received Type r.Options in Create and set QoS: %+v", req.Qos)
				break
			}
		}
	}

	req.TotalSize = vsz
	req.AccountID = d.TenantID
	req.Name = r.Name
	_, err = d.Client.CreateVolume(&req)
	if err != nil {
		return err
	}
	return  nil
}

func (d SolidFireDriver) Remove(r *volume.RemoveRequest) error {
	log.Info("Remove/Delete Volume: ", r.Name)
	v, err := d.Client.GetVolumeByName(r.Name, d.TenantID)
	if err != nil {
		log.Error("Failed to retrieve volume named ", r.Name, "during Remove operation: ", err)
		return err
	}
	d.Client.DetachVolume(v)
	err = d.Client.DeleteVolume(v.VolumeID)
	if err != nil {
		// FIXME(jdg): Check if it's a "DNE" error in that case we're golden
		log.Error("Error encountered during delete: ", err)
	}
	return nil
}

func (d SolidFireDriver) Path(r *volume.PathRequest) (*volume.PathResponse, error) {
	log.Info("Retrieve path info for volume: ", r.Name)
	path := filepath.Join(d.MountPoint, r.Name)
	log.Debug("Path reported as: ", path)
	return &volume.PathResponse{Mountpoint: path}, nil
}

func (d SolidFireDriver) Mount(r *volume.MountRequest) (*volume.MountResponse, error) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	log.Infof("Mounting volume %s on %s\n", r.Name, "solidfire")
	v, err := d.Client.GetVolumeByName(r.Name, d.TenantID)
	if err != nil {
		log.Errorf("Failed to retrieve volume by name in mount operation: ", r.Name)
		return &volume.MountResponse{}, err
	}
	path, device, err := d.Client.AttachVolume(&v, d.InitiatorIFace)
	if path == "" || device == "" && err == nil {
		log.Error("Missing path or device, but err not set?")
		log.Debug("Path: ", path, ",Device: ", device)
		return &volume.MountResponse{}, err

	}
	if err != nil {
		log.Errorf("Failed to perform iscsi attach of volume %s: %v", r.Name, err)
		return &volume.MountResponse{}, err
	}
	log.Debugf("Attached volume at (path, devfile): %s, %s", path, device)
	if sfapi.GetFSType(device) == "" {
		//TODO(jdg): Enable selection of *other* fs types
		err := sfapi.FormatVolume(device, "ext4")
		if err != nil {
			log.Errorf("Failed to format device: ", device)
		    return &volume.MountResponse{}, err
		}
	}
	if sfapi.Mount(device, d.MountPoint+"/"+r.Name) != nil {
		log.Error("Failed to mount volume: ", r.Name)
		    return &volume.MountResponse{}, err
	}
	return &volume.MountResponse{Mountpoint: d.MountPoint + "/" + r.Name}, nil
}

func (d SolidFireDriver) Unmount(r *volume.UnmountRequest) error {
	log.Info("Unmounting volume: ", r.Name)
	sfapi.Umount(filepath.Join(d.MountPoint, r.Name))
	v, err := d.Client.GetVolumeByName(r.Name, d.TenantID)
	if err != nil {
		return err
	}
	d.Client.DetachVolume(v)
	return nil
}

func (d SolidFireDriver) Get(r *volume.GetRequest) (*volume.GetResponse, error) {
	log.Info("Get volume: ", r.Name)
	path := filepath.Join(d.MountPoint, r.Name)
	v, err := d.Client.GetVolumeByName(r.Name, d.TenantID)
	if err != nil {
		log.Error("Failed to retrieve volume named ", r.Name, "during Get operation: ", err)
		return &volume.GetResponse{}, err
	}
	return &volume.GetResponse{Volume: &volume.Volume{Name: v.Name, Mountpoint: path}}, nil
}

func (d SolidFireDriver) List() (*volume.ListResponse, error) {
	log.Info("List Volumes")
	var vols []*volume.Volume
	var req sfapi.ListVolumesForAccountRequest
	req.AccountID = d.TenantID
	vlist, err := d.Client.ListVolumesForAccount(&req)
	if err != nil {
		log.Error("Failed to retrieve volume list:", err)
		return &volume.ListResponse{}, err
	}

	for _, v := range vlist {
		if v.Status == "active" && v.AccountID == d.TenantID {
	        path := filepath.Join(d.MountPoint, v.Name)
			vols = append(vols, &volume.Volume{Name: v.Name, Mountpoint: path})
		}
	}
	return &volume.ListResponse{Volumes: vols}, nil
}
