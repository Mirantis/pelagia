/*
Copyright 2025 Mirantis IT.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package diskdaemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/pkg/errors"
)

var (
	// but we are interseted only in next fields: NAME,KNAME,MAJ:MIN,ROTA,TYPE,PKNAME
	lsblkCmd         = "lsblk -J -p -O"
	udevadmInfoCmd   = "udevadm info -r --query=%s %s"
	activeLvsCmd     = "lvm lvs --reportformat json -o lv_dm_path"
	activateAllLvCmd = "pvscan --cache %s"
	// mock for testing
	runShellCmd = execCmd
)

type DeviceLsblk struct {
	Name       string        `json:"name"`
	Kname      string        `json:"kname"`
	Serial     string        `json:"serial,omitempty"`
	MajMin     string        `json:"maj:min"`
	Rotational bool          `json:"rota"`
	Type       string        `json:"type"`
	Parent     string        `json:"pkname,omitempty"`
	Childrens  []DeviceLsblk `json:"children,omitempty"`
	FsType     string        `json:"fstype,omitempty"`
}

type LsblkReport struct {
	Blockdevices []DeviceLsblk `json:"blockdevices"`
}

type LvmReport struct {
	Report []LvmReportField `json:"report,omitempty"`
}

type LvmReportField struct {
	Lvs []LvReport `json:"lv,omitempty"`
}

type LvReport struct {
	LvDmPath string `json:"lv_dm_path"`
}

func getLsblk() (*LsblkReport, error) {
	stdOut, stdErr, err := runShellCmd(lsblkCmd)
	if err != nil {
		log.Error().Msg(stdErr)
		return nil, err
	}
	var lsblk LsblkReport
	err = json.Unmarshal([]byte(stdOut), &lsblk)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal lsblk output")
	}
	return &lsblk, nil
}

func getUdevadmInfo(deviceName, query string) (string, error) {
	if query == "" {
		query = "all"
	}
	cmd := fmt.Sprintf(udevadmInfoCmd, query, deviceName)
	stdOut, stdErr, err := runShellCmd(cmd)
	if stdErr != "" {
		log.Error().Msg(stdErr)
	}
	return stdOut, err
}

func getActiveLvs() ([]string, error) {
	stdOut, stdErr, err := runShellCmd(activeLvsCmd)
	if err != nil {
		log.Error().Msg(stdErr)
		return nil, err
	}
	var lvmReport LvmReport
	err = json.Unmarshal([]byte(stdOut), &lvmReport)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal lvm lvs output")
	}
	var activeLvs []string
	for _, report := range lvmReport.Report {
		activeLvs = make([]string, len(report.Lvs))
		for idx, lvReport := range report.Lvs {
			activeLvs[idx] = lvReport.LvDmPath
		}
	}
	return activeLvs, nil
}

func cacheLvms(newLvms map[string]bool) error {
	for disk := range newLvms {
		log.Info().Msgf("Caching logical volume table on '%s'", disk)
		stdOut, stdErr, err := runShellCmd(fmt.Sprintf(activateAllLvCmd, disk))
		if stdOut != "" {
			log.Info().Msg(stdOut)
		}
		if err != nil {
			if stdErr != "" {
				log.Error().Msg(stdErr)
			}
			return errors.Wrap(err, "failed to cache volumes")
		}
	}
	return nil
}

type OsdVolumeInfo struct {
	Devices []string      `json:"devices"`
	LvPath  string        `json:"lv_path"`
	Path    string        `json:"path"`
	Tags    OsdVolumeTags `json:"tags"`
	Type    string        `json:"type"`
}

type OsdVolumeTags struct {
	BlockDevice string `json:"ceph.block_device"`
	DBDevice    string `json:"ceph.db_device"`
	ClusterFSID string `json:"ceph.cluster_fsid"`
	OsdFSID     string `json:"ceph.osd_fsid"`
}

func getCephVolumeLvmList() (map[string][]OsdVolumeInfo, error) {
	cmd := "ceph-volume lvm list --format json"
	var cephVolumeLvm map[string][]OsdVolumeInfo
	stdOut, stdErr, err := runShellCmd(cmd)
	if err != nil {
		if stdErr != "" {
			log.Error().Msg(stdErr)
		}
		return nil, err
	}
	err = json.Unmarshal([]byte(stdOut), &cephVolumeLvm)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse output for command '%s'", cmd)
	}
	return cephVolumeLvm, nil
}

func execCmd(command string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
