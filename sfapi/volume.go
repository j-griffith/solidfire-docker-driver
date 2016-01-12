package sfapi

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"strings"
)

func (c *Client) ListActiveVolumes(listVolReq *ListActiveVolumesRequest) (volumes []Volume, err error) {
	response, err := c.Request("ListActiveVolumes", listVolReq, newReqID())
	if err != nil {
		log.Error(err)
		return
	}
	var result ListVolumesResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Fatal(err)
		return nil, err
	}
	volumes = result.Result.Volumes
	return
}

func (c *Client) GetVolume(sfID int64, sfName string) (v Volume, err error) {
	var listReq ListActiveVolumesRequest

	volumes, err := c.ListActiveVolumes(&listReq)
	if err != nil {
		fmt.Println("Error retrieving volumes")
		return Volume{}, err
	}
	for _, vol := range volumes {
		if sfID == vol.VolumeID {
			log.Debugf("Found volume by ID: %v", vol)
			v = vol
			break
		} else if sfName != "" && sfName == vol.Name {
			log.Debugf("Found volume by Name: %v", vol)
			v = vol
			break
		}
	}
	return
}

func (c *Client) CloneVolume(req *CloneVolumeRequest) (vol Volume, err error) {
	response, err := c.Request("CloneVolume", req, newReqID())
	var result CloneVolumeResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Fatal(err)
		return Volume{}, err
	}
	vol, err = c.GetVolume(result.Result.VolumeID, "")
	return
}

func (c *Client) CreateVolume(createReq *CreateVolumeRequest) (vol Volume, err error) {
	response, err := c.Request("CreateVolume", createReq, newReqID())
	if err != nil {
		return Volume{}, err
	}
	var result CreateVolumeResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Fatal(err)
		return Volume{}, err
	}

	vol, err = c.GetVolume(result.Result.VolumeID, "")
	return
}

func (c *Client) AddVolumeToAccessGroup(groupID int64, volIDs []int64) (err error) {
	req := &AddVolumesToVolumeAccessGroupRequest{
		VolumeAccessGroupID: groupID,
		Volumes:             volIDs,
	}
	_, err = c.Request("AddVolumesToVolumeAccessGroup", req, newReqID())
	if err != nil {
		log.Error("Failed to add volume(s) to VAG %d: ", groupID)
		return err
	}
	return err
}

func (c *Client) DeleteRange(startID, endID int64) {
	idx := startID
	for idx < endID {
		c.DeleteVolume(idx)
	}
	return
}

func (c *Client) DeleteVolume(volumeID int64) (err error) {
	// TODO(jdg): Add options like purge=True|False, range, ALL etc
	var req DeleteVolumeRequest
	req.VolumeID = volumeID
	_, err = c.Request("DeleteVolume", req, newReqID())
	if err != nil {
		log.Error("Failed to delete volume ID: ", volumeID)
		return err
	}
	return
}

func (c *Client) DetachVolume(volumeID int64, name string) (err error) {
	if c.SVIP == "" {
		err = errors.New("Unable to perform iSCSI actions without setting SVIP")
		return
	}

	v, err := c.GetVolume(volumeID, name)
	if err != nil {
		err = errors.New("Failed to find volume for detach")
		return
	}
	tgt := &ISCSITarget{
		Ip:     c.SVIP,
		Portal: c.SVIP,
		Iqn:    v.Iqn,
	}
	err = iscsiDisableDelete(tgt)
	return
}

func (c *Client) AttachVolume(v *Volume) (string, string, error) {
	if c.SVIP == "" {
		err := errors.New("Unable to perform iSCSI actions without setting SVIP")
		return "", "", err
	}
	path := "/dev/disk/by-path/ip-"
	if iscsiSupported() == false {
		err := errors.New("Unable to attach, open-iscsi tools not found on host")
		return "", "", err
	}

	path = path + c.SVIP + "-iscsi-" + v.Iqn + "-lun-0"
	device := getDeviceFileFromIscsiPath(path)
	if waitForPathToExist(path, 1) {
		return path, device, nil
	}

	targets, err := iscsiDiscovery(c.SVIP)
	if err != nil {
		log.Error("Failure encountered during iSCSI Discovery: ", err)
		log.Error("Have you setup the Volume Access Group?")
		err = errors.New("iSCSI Discovery failed")
		return "", "", err
	}

	if len(targets) < 1 {
		log.Warning("Discovered zero targets at: ", c.SVIP)
		return "", "", err
	}

	tgt := ISCSITarget{}
	for _, t := range targets {
		if strings.Contains(t, v.Iqn) {
			tgt.Discovery = t
		}
	}
	if tgt.Discovery == "" {
		log.Error("Failed to discover requested target: ", v.Iqn, " on: ", c.SVIP)
		return "", "", err
	}
	log.Debug("Discovered target: ", tgt.Discovery)

	parsed := strings.FieldsFunc(tgt.Discovery, func(r rune) bool {
		return r == ',' || r == ' '
	})

	tgt.Ip = parsed[0]
	tgt.Iqn = parsed[2]
	err = iscsiLogin(&tgt)
	if err != nil {
		log.Error("Failed to connect to iSCSI target (", tgt.Iqn, ")")
		return "", "", err
	}
	if waitForPathToExist(path, 10) == false {
		log.Error("Failed to find connection after 10 seconds")
		return "", "", err
	}

	device = strings.TrimSpace(getDeviceFileFromIscsiPath(path))
	return path, device, nil
}
