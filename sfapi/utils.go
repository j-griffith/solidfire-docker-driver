package sfapi

import (
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
	"time"
)

func waitForPathToExist(fileName string, numTries int) bool {
	log.Debug("Check for presence of: ", fileName)
	for i := 0; i < numTries; i++ {
		_, err := os.Stat(fileName)
		if err == nil {
			return true
		}
		if err != nil && !os.IsNotExist(err) {
			return false
		}
		time.Sleep(time.Second)
	}
	return false
}

func getDeviceFileFromIscsiPath(iscsiPath string) (devFile string) {
	out, err := exec.Command("sudo", "ls", "-la", iscsiPath).Output()
	if err != nil {
		return
	}
	d := strings.Split(string(out), "../../")
	devFile = "/dev/" + d[1]
	devFile = strings.TrimSpace(devFile)
	return
}

func iscsiSupported() bool {
	_, err := exec.Command("iscsiadm", "-h").Output()
	if err != nil {
		log.Debug("iscsiadm tools not found on this host")
		return false
	}
	return true
}

func iscsiDiscovery(portal string) (targets []string, err error) {
	log.Debug("Issue sendtargets: sudo iscsiadm -m discovery -t sendtargets -p ", portal)
	out, err := exec.Command("sudo", "iscsiadm", "-m", "discovery", "-t", "sendtargets", "-p", portal).Output()
	if err != nil {
		log.Error("Error encountered in sendtargets cmd: ", out)
		return
	}
	targets = strings.Split(string(out), "\n")
	return

}

func iscsiLogin(tgt *ISCSITarget) (err error) {
	log.Debugf("Attempt to login to iSCSI target: %v", tgt)
	_, err = exec.Command("sudo", "iscsiadm", "-m", "node", "-p", tgt.Ip, "-T", tgt.Iqn, "--login").Output()
	if err != nil {
		log.Errorf("Received error on login attempt: %v", err)
	}
	return err
}

func iscsiDisableDelete(tgt *ISCSITarget) (err error) {
	_, err = exec.Command("sudo", "iscsiadm", "-m", "node", "-T", tgt.Iqn, "--portal", tgt.Ip, "-u").Output()
	if err != nil {
		return
	}
	_, err = exec.Command("sudo", "iscsiadm", "-m", "node", "-o", "delete", "-T", tgt.Iqn).Output()
	return
}

func getFSType(device string) string {
	out, err := exec.Command("blkid", "/dev/sdd").Output()
	if err != nil {
		return ""
	}
	result := strings.Split(string(out), " ")
	for _, v := range result {
		if v == "TYPE=\"ext4\"" {
			return "ext4"
		} else if v == "TYPE=\"xfs\"" {
			return "xfs"
		} else {
			return ""
		}
	}
	return ""
}

func formatVolume(device string, fsType string, overwrite bool) error {
	// First let's make sure it doesn't already have an FS on it
	cmd := "mkfs.ext4"
	if fsType == "xfs" {
		cmd = "mkfs.xfs"
	}
	out, err := exec.Command(cmd, "-F", device).Output()
	log.Debugf("Result of mkfs cmd: ", out)
	return err
}
