package sfapi

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
)

func (c *Client) CreateVolumeAccessGroup(r *CreateVolumeAccessGroupRequest) (vagID int64, err error) {
	var result CreateVolumeAccessGroupResult
	response, err := c.Request("CreateVolumeAccessGroup", r, newReqID())
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Fatal(err)
		return 0, err
	}
	vagID = result.Result.VagID
	return

}

func (c *Client) ListVolumeAccessGroups(r *ListVolumeAccessGroupsRequest) (vags []VolumeAccessGroup, err error) {
	response, err := c.Request("ListVolumeAccessGroups", r, newReqID())
	if err != nil {
		log.Error(err)
		return
	}
	var result ListVolumesAccessGroupsResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Fatal(err)
		return nil, err
	}
	vags = result.Result.Vags
	return
}

func (c *Client) AddInitiatorsToVolumeAccessGroup(r *AddInitiatorsToVolumeAccessGroupRequest) error {
	response, err := c.Request("AddInitiatorsToVolumeAccessGroup", r, newReqID())
	if err != nil {
		log.Error(string(response))
		log.Error(err)
		return err
	}
	return nil
}
