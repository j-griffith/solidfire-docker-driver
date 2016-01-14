package sfcli

import (
	"errors"
	"fmt"

	"github.com/alecthomas/units"
	"github.com/codegangsta/cli"
	"github.com/j-griffith/solidfire-docker-driver/sfapi"
	"strconv"
	"strings"
)

var (
	volumeCmd = cli.Command{
		Name:  "volume",
		Usage: "volume related commands",
		Subcommands: []cli.Command{
			volumeCreateCmd,
			volumeDeleteCmd,
			volumeListCmd,
			volumeAttachCmd,
			volumeDetachCmd,
			volumeAddToVag,
		},
	}
	volumeCreateCmd = cli.Command{
		Name:  "create",
		Usage: "create a new volume: `create [options] NAME`",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "size",
				Usage: "size of volume in bytes or GiB: `[--size 1073741824|1GiB]`",
			},
			cli.StringFlag{
				Name:  "account",
				Usage: "account id to assign volume: `[--account 488]`",
			},
			cli.StringFlag{
				Name:  "qos",
				Usage: "min,max and burst qos settings for new volume: `[--qos 1000,5000,15000]`",
			},
			cli.StringFlag{
				Name:  "vag",
				Usage: "Volume Access Group to add volume to on create: `[--vag 8]`",
			},
		},
		Action: cmdVolumeCreate,
	}

	volumeDeleteCmd = cli.Command{
		Name:  "delete",
		Usage: "delete an existing volume: `delete VOLUME-ID`",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "range",
				Value: "",
				Usage: ": deletes a range of volume ID's `[--range <startID-endID>]`",
			},
		},
		Action: cmdVolumeDelete,
	}

	volumeListCmd = cli.Command{
		Name:   "list",
		Usage:  "list existing volumes: `list`",
		Action: cmdVolumeList,
	}

	volumeAttachCmd = cli.Command{
		Name:  "attach",
		Usage: "iscsi attach volume to host (requires permissions and iscsiadm): `attach VOLUME-ID`",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "iface",
				Usage: "Device file of iSCSI interface (initiator): `[--iface /dev/p1p2]`",
			},
		},
		Action: cmdVolumeAttach,
	}

	volumeDetachCmd = cli.Command{
		Name:   "detach",
		Usage:  "iscsi detach volume from host (requires permissions and iscsiadm): `detach VOLUME-ID`",
		Action: cmdVolumeDetach,
	}

	volumeAddToVag = cli.Command{
		Name:   "addtovag",
		Usage:  "Add existing Volume to existing Volume Access Group: `addtovag VOLUME-ID VAG-ID`",
		Action: cmdVolumeAddToVag,
	}
)

func cmdVolumeAttach(c *cli.Context) {
	id := c.Args().First()
	volID, _ := strconv.ParseInt(id, 10, 64)
	v, err := client.GetVolume(volID, "")
	if err != nil {
		err = errors.New("Failed to find volume for attach")
		return
	}
	netDev := c.String("iface")
	if c.String("iface") == "" {
		netDev = "default"
	}
	path, device, err := client.AttachVolume(&v, netDev)
	if err != nil {
		fmt.Println("Error encountered while performing iSCSI attach on Volume: ", volID)
		fmt.Println(err)
		return

	}

	fmt.Println("-------------------------------------------")
	fmt.Println("Succesfully iSCSI Attached Volume:")
	fmt.Println("-------------------------------------------")
	fmt.Println("ID:         ", volID)
	fmt.Println("Path:       ", path)
	fmt.Println("Device:     ", device)
	fmt.Println("-------------------------------------------")

}

func cmdVolumeDetach(c *cli.Context) {
	id := c.Args().First()
	volID, _ := strconv.ParseInt(id, 10, 64)
	v, err := client.GetVolume(volID, "")
	err = client.DetachVolume(v)
	if err != nil {
		fmt.Println("Error encountered while performing iSCSI detach of Volume: ", volID)
		fmt.Println(err)
		return

	}
}

func cmdVolumeAddToVag(c *cli.Context) {
	vID, _ := strconv.ParseInt(c.Args().First(), 10, 64)
	vagID, _ := strconv.ParseInt(c.Args()[1], 10, 64)

	var volIDs []int64
	volIDs = append(volIDs, vID)
	err := client.AddVolumeToAccessGroup(vagID, volIDs)
	if err != nil {
		fmt.Printf("Failed to add volume to VAG ID: %d\n", vagID)
		return
	}
	fmt.Printf("Succesfully added volume to VAG ID: %d\n", vagID)
}

func cmdVolumeCreate(c *cli.Context) {
	var req sfapi.CreateVolumeRequest
	var qos sfapi.QoS
	req.Name = c.Args().First()

	sz := int64(0)
	if c.String("size") == "" && client.DefaultVolSize != 0 {
		sz = client.DefaultVolSize
	} else if c.String("size") != "" {
		sz, _ = units.ParseStrictBytes(c.String("size"))
	} else {
		fmt.Println("You must specify size for volumeCreate")
		return
	}

	account := int64(0)
	if c.String("account") == "" && client.DefaultAccountID != 0 {
		account = client.DefaultAccountID
	} else if c.String("size") != "" {
		account, _ = units.ParseStrictBytes(c.String("account"))
	} else {
		fmt.Println("You must specify an account for volumeCreate")
		return
	}

	req.TotalSize = sz
	req.AccountID = account
	if c.String("qos") != "" {
		iops := strings.Split(c.String("qos"), ",")
		qos.MinIOPS, _ = strconv.ParseInt(iops[0], 10, 64)
		qos.MaxIOPS, _ = strconv.ParseInt(iops[2], 10, 64)
		qos.BurstIOPS, _ = strconv.ParseInt(iops[2], 10, 64)
		req.Qos = qos
	}

	v, _ := client.CreateVolume(&req)
	fmt.Println("-------------------------------------------")
	fmt.Println("Succesfully Created Volume:")
	fmt.Println("-------------------------------------------")
	fmt.Println("ID:         ", v.VolumeID)
	fmt.Println("Name:       ", v.Name)
	fmt.Println("Size (GiB): ", v.TotalSize/int64(units.GiB))
	fmt.Println("QoS :       ", "minIOPS:", v.Qos.MinIOPS, "maxIOPS:", v.Qos.MaxIOPS, "burstIOPS:", v.Qos.BurstIOPS)
	fmt.Println("Account:    ", v.AccountID)
	fmt.Println("-------------------------------------------")

	if c.String("vag") != "" {
		vagID, _ := strconv.ParseInt(c.String("vag"), 10, 64)
		var volIDs []int64
		volIDs = append(volIDs, v.VolumeID)
		err := client.AddVolumeToAccessGroup(vagID, volIDs)
		if err != nil {
			fmt.Printf("Failed to add volume to VAG ID: %d\n", vagID)
			return
		}
		fmt.Printf("Succesfully added volume to VAG ID: %d\n", vagID)
	}
	return
}

func cmdVolumeDelete(c *cli.Context) {
	volumes := c.String("range")
	if volumes != "" {
		ids := strings.Split(volumes, "-")
		fmt.Println("You've selected to delete volumes: ", ids[0], " through ", ids[1])
		fmt.Print("Are you sure you want to do this [yes/no]: ")
		if confirm() {
			startID, _ := strconv.ParseInt(ids[0], 10, 64)
			endID, _ := strconv.ParseInt(ids[1], 10, 64)
			client.DeleteRange(startID, endID)
		}

	} else {
		for _, arg := range c.Args() {
			vID, _ := strconv.ParseInt(arg, 10, 64)
			client.DeleteVolume(vID)
		}
	}
}

func cmdVolumeList(c *cli.Context) {
	var req sfapi.ListActiveVolumesRequest
	volumes, err := client.ListActiveVolumes(&req)
	if err != nil {
		fmt.Println(err)
	} else {
		printVolList(volumes)
	}

}
