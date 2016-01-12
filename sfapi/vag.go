package sfapi

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
)

func (c *Client) CreateVolumeAccessGroup(createVagReq *CreateVolumeAccessGroupRequest) (vagID int64, err error) {
	var result CreateVolumeAccessGroupResult
	response, err := c.Request("CreateVolumeAccessGroup", createVagReq, newReqID())
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Fatal(err)
		return 0, err
	}
	vagID = result.Result.VagID
	return

}

func (c *Client) ListVolumeAccessGroups(listVAGReq *ListVolumeAccessGroupsRequest) (vags []VolumeAccessGroup, err error) {
	response, err := c.Request("ListVolumeAccessGroups", listVAGReq, newReqID())
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
