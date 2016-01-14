package sfapi

import (
	"encoding/json"
	"errors"
	log "github.com/Sirupsen/logrus"
	"strings"
)

func (c *Client) ListActiveVolumes(listVolReq *ListActiveVolumesRequest) (volumes []Volume, err error) {
	response, err := c.Request("ListActiveVolumes", listVolReq, newReqID())
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var result ListVolumesResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Fatal(err)
		return nil, err
	}
	volumes = result.Result.Volumes
	return nil, err
}

func (c *Client) GetVolume(sfID int64, sfName string) (v Volume, err error) {
	var listReq ListActiveVolumesRequest
	volumes, err := c.ListActiveVolumes(&listReq)
	if err != nil {
		log.Error("Error retrieving volumes: ", err)
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
	return v, err
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

func (c *Client) DetachVolume(v Volume) (err error) {
	if c.SVIP == "" {
		err = errors.New("Unable to perform iSCSI actions without setting SVIP")
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

func (c *Client) AttachVolume(v *Volume, iface string) (path, device string, err error) {
	var req GetAccountByIDRequest
	path = "/dev/disk/by-path/ip-" + c.SVIP + "-iscsi-" + v.Iqn + "-lun-0"

	if c.SVIP == "" {
		err = errors.New("Unable to perform iSCSI actions without setting SVIP")
		log.Error(err)
		return path, device, err
	}

	if iscsiSupported() == false {
		err := errors.New("Unable to attach, open-iscsi tools not found on host")
		log.Error(err)
		return path, device, err
	}

	req.AccountID = v.AccountID
	a, err := c.GetAccountByID(&req)
	if err != nil {
		log.Error("Failed to get account ", v.AccountID, ": ", err)
		return path, device, err
	}

	// Make sure it's not already attached
	if waitForPathToExist(path, 1) {
		log.Debug("Get device file from path: ", path)
		device = strings.TrimSpace(getDeviceFileFromIscsiPath(path))
		return path, device, nil
	}

	err = LoginWithChap(v.Iqn, c.SVIP, a.Username, a.InitiatorSecret, iface)
	if err != nil {
		log.Error(err)
		return path, device, err
	}
	if waitForPathToExist(path, 5) {
		device = strings.TrimSpace(getDeviceFileFromIscsiPath(path))
		return path, device, nil
	}
	return path, device, nil
}
