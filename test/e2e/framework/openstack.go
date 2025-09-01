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

package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CinderListItem struct {
	Name   string `json:"Name,omitempty"`
	Status string `json:"Status,omitempty"`
	Size   int    `json:"Size,omitempty"`
}

type CinderShow struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Status      string `json:"status,omitempty"`
	Size        int    `json:"size,omitempty"`
	Type        string `json:"type,omitempty"`
	Attachments []struct {
		VolumeID string `json:"volume_id,omitempty"`
		ServerID string `json:"server_id,omitempty"`
		Device   string `json:"device,omitempty"`
	} `json:"attachments,omitempty"`
}

type CinderTypeListItem struct {
	Name string `json:"Name,omitempty"`
}

type NovaListItem struct {
	Name   string `json:"Name,omitempty"`
	Status string `json:"Status,omitempty"`
}

type NovaShow struct {
	ID        string              `json:"id,omitempty"`
	Name      string              `json:"name,omitempty"`
	Status    string              `json:"status,omitempty"`
	Flavor    string              `json:"flavor,omitempty"`
	Image     string              `json:"image,omitempty"`
	Power     int                 `json:"OS-EXT-STS:power_state,omitempty"`
	Host      string              `json:"OS-EXT-SRV-ATTR:host"`
	Keypair   string              `json:"key_name,omitempty"`
	Addresses map[string][]string `json:"addresses,omitempty"`
	Volumes   []struct {
		ID string `json:"id,omitempty"`
	} `json:"volumes_attached,omitempty"`
}

type GlanceListItem struct {
	Name   string `json:"Name,omitempty"`
	Status string `json:"Status,omitempty"`
}

type GlanceShow struct {
	Name      string `json:"name,omitempty"`
	Status    string `json:"status,omitempty"`
	Size      int    `json:"size,omitempty"`
	Locations []struct {
		URL string `json:"url,omitempty"`
	} `json:"locations,omitempty"`
}

type SecurityGroupListItem struct {
	Name string `json:"Name,omitempty"`
	ID   string `json:"ID,omitempty"`
}

type PortListItem struct {
	ID       string `json:"ID,omitempty"`
	FixedIPs []struct {
		IP string `json:"ip_address,omitempty"`
	} `json:"Fixed IP Addresses,omitempty"`
	Status string `json:"Status,omitempty"`
}

type NovaEventListItem struct {
	ID        string `json:"Request ID,omitempty"`
	Action    string `json:"Action,omitempty"`
	StartTime string `json:"Start Time,omitempty"`
}

type NovaEventShow struct {
	Events []struct {
		Event      string `json:"event,omitempty"`
		StartTime  string `json:"start_time,omitempty"`
		FinishTime string `json:"finish_time,omitempty"`
		Result     string `json:"result,omitempty"`
	} `json:"events,omitempty"`
}

type ManilaExportLocationItem struct {
	ID        string `json:"ID,omitempty"`
	Path      string `json:"Path,omitempty"`
	Preferred bool   `json:"Preferred,omitempty"`
}

type ManilaShareAccessItem struct {
	ID        string `json:"ID,omitempty"`
	AccessKey string `json:"Access Key,omitempty"`
	AccessTo  string `json:"Access To,omitempty"`
}

type ManilaConfigParams struct {
	MonEndpoints     string
	CephFsName       string
	CephFsClientName string
	CephFsClientKey  string
}

type StackShow struct {
	Name   string `json:"stack_name,omitempty"`
	Status string `json:"stack_status,omitempty"`
}

type StackOutputShow struct {
	Key   string `json:"output_key,omitempty"`
	Value string `json:"output_value,omitempty"`
}

type OpenstackClient struct {
	KeystonePod *v1.Pod
	Container   string
}

func (c *ManagedConfig) OpenstackClientSet() error {
	pod, containerName, err := c.GetKeystonePod()
	if err != nil {
		return err
	}
	c.OpenstackClient = &OpenstackClient{
		KeystonePod: pod,
		Container:   containerName,
	}
	return nil
}

func (c *ManagedConfig) GetKeystonePod() (*v1.Pod, string, error) {
	pod, err := c.GetPodByLabel("openstack", "application=keystone,component=client")
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to find keystone-client pod")
	}
	return pod, pod.Spec.Containers[0].Name, nil
}

func (c *ManagedConfig) RunOpenstackCommand(command string) (string, string, error) {
	if c.OpenstackClient == nil {
		err := c.OpenstackClientSet()
		if err != nil {
			return "", "", err
		}
	}
	stdout, stderr, err := c.RunPodCommand(command, c.OpenstackClient.Container, c.OpenstackClient.KeystonePod)
	if err != nil {
		errMsg := fmt.Sprintf("openstack cli command '%s' failed", command)
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %v)", stderr)
		}
		return "", stderr, errors.New(errMsg)
	}
	return stdout, stderr, err
}

func (c *ManagedConfig) CinderVolumeList() ([]CinderListItem, error) {
	stdout, _, err := c.RunOpenstackCommand("openstack volume list -f json")
	if err != nil {
		return nil, errors.Wrap(err, "failed run 'cinder volume list'")
	}
	result := []CinderListItem{}
	err = json.Unmarshal([]byte(stdout), &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse output for 'cinder volume list'")
	}
	return result, nil
}

func (c *ManagedConfig) NovaServerList() ([]NovaListItem, error) {
	stdout, _, err := c.RunOpenstackCommand("openstack server list -f json")
	if err != nil {
		return nil, errors.Wrap(err, "failed run 'nova server list'")
	}
	result := []NovaListItem{}
	err = json.Unmarshal([]byte(stdout), &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse output for 'nove server list'")
	}
	return result, nil
}

func (c *ManagedConfig) GlanceImageList() ([]GlanceListItem, error) {
	stdout, _, err := c.RunOpenstackCommand("openstack image list -f json")
	if err != nil {
		return nil, errors.Wrap(err, "failed run 'glance images list'")
	}
	result := []GlanceListItem{}
	err = json.Unmarshal([]byte(stdout), &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse output for 'glance images list'")
	}
	return result, nil
}

func (c *ManagedConfig) CinderVolumeTypeList() ([]CinderTypeListItem, error) {
	stdout, _, err := c.RunOpenstackCommand("openstack volume type list -f json")
	if err != nil {
		return nil, errors.Wrap(err, "failed run 'cinder volume type list'")
	}
	result := []CinderTypeListItem{}
	err = json.Unmarshal([]byte(stdout), &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse output for 'cinder volume type list'")
	}
	return result, nil
}

func (c *ManagedConfig) CinderVolumeShow(name string) (*CinderShow, error) {
	stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack volume show %s -f json", name))
	if err != nil {
		return nil, errors.Wrapf(err, "failed run 'openstack volume show %s'", name)
	}
	result := CinderShow{}
	err = json.Unmarshal([]byte(stdout), &result)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse output for 'cinder volume show %s'", name)
	}
	return &result, nil
}

func (c *ManagedConfig) NovaServerShow(name string) (*NovaShow, error) {
	stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server show %s -f json", name))
	if err != nil {
		return nil, errors.Wrapf(err, "failed run 'openstack server show %s'", name)
	}
	result := NovaShow{}
	err = json.Unmarshal([]byte(stdout), &result)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse output for 'nova server show %s'", name)
	}
	return &result, nil
}

func (c *ManagedConfig) GlanceImageShow(name string) (*GlanceShow, error) {
	stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack image show %s -f json", name))
	if err != nil {
		return nil, errors.Wrapf(err, "failed run 'openstack image show %s'", name)
	}
	result := GlanceShow{}
	err = json.Unmarshal([]byte(stdout), &result)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse output for 'glance image show %s'", name)
	}
	return &result, nil
}

func (c *ManagedConfig) CinderVolumeDelete(name string, waiting bool) error {
	err := wait.PollUntilContextTimeout(c.Context, 15*time.Second, 5*time.Minute, true, func(_ context.Context) (done bool, err error) {
		_, stderr, err := c.RunOpenstackCommand(fmt.Sprintf("openstack volume delete %s", name))
		if !waiting {
			return true, nil
		}
		if err != nil && strings.Contains(stderr, fmt.Sprintf("No volume with a name or ID of '%s' exists.", name)) {
			return true, nil
		}
		TF.Log.Info().Msgf("'openstack volume delete %s' is not complete yet", name)
		return false, nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to wait for 'openstack volume delete %s' completion", name)
	}
	return nil
}

func (c *ManagedConfig) NovaServerDelete(name string, waiting bool) error {
	err := wait.PollUntilContextTimeout(c.Context, 15*time.Second, 5*time.Minute, true, func(_ context.Context) (done bool, err error) {
		_, stderr, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server delete %s", name))
		if !waiting {
			return true, nil
		}
		if err != nil && strings.Contains(stderr, fmt.Sprintf("No server with a name or ID of '%s' exists.", name)) {
			return true, nil
		}
		TF.Log.Info().Msgf("'openstack server delete %s' is not complete yet", name)
		return false, nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to wait for 'openstack server delete %s' completion", name)
	}
	return nil
}

func (c *ManagedConfig) GlanceImageDelete(name string, waiting bool) error {
	err := wait.PollUntilContextTimeout(c.Context, 15*time.Second, 5*time.Minute, true, func(_ context.Context) (done bool, err error) {
		_, stderr, err := c.RunOpenstackCommand(fmt.Sprintf("openstack image delete %s", name))
		if !waiting {
			return true, nil
		}
		if err != nil && strings.Contains(stderr, fmt.Sprintf("Failed to delete image with name or ID '%s': No Image found for %s", name, name)) {
			return true, nil
		}
		TF.Log.Info().Msgf("'openstack image delete %s' is not complete yet", name)
		return false, nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to wait for 'openstack image delete %s' completion", name)
	}
	return nil
}

func (c *ManagedConfig) CinderVolumeCreate(name string, size int, volumeType string, waiting bool) (*CinderShow, error) {
	command := fmt.Sprintf("openstack volume create %s --size %d", name, size)
	if volumeType != "" {
		command += fmt.Sprintf(" --type %s", volumeType)
	}
	_, _, err := c.RunOpenstackCommand(command)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run '%s'", command)
	}
	result := CinderShow{}
	err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 5*time.Minute, true, func(_ context.Context) (done bool, err error) {
		stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack volume show %s -f json", name))
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed run 'openstack volume show %s'", name)
			return false, nil
		}
		result = CinderShow{}
		err = json.Unmarshal([]byte(stdout), &result)
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to parse output for 'openstack volume show %s'", name)
			return false, nil
		}
		if !waiting {
			return true, nil
		}
		return result.Status == "available" || result.Status == "in-use", nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to wait for 'openstack volume create %s' completion", name)
	}
	return &result, nil
}

func (c *ManagedConfig) NovaServerCreate(name, flavor, keypair, image, network string, waiting bool) (*NovaShow, error) {
	// get default security group from admin project
	stdout, _, err := c.RunOpenstackCommand("openstack security group list --project admin -f json")
	if err != nil {
		return nil, errors.Wrap(err, "failed to run 'openstack security group list --project admin'")
	}
	secGroup := []SecurityGroupListItem{}
	err = json.Unmarshal([]byte(stdout), &secGroup)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse output for 'openstack security group list --project admin'")
	}

	// create server
	command := fmt.Sprintf("openstack server create %s --flavor %s --image %s --security-group %s --key-name %s --network %s", name, flavor, image, secGroup[0].ID, keypair, network)
	_, _, err = c.RunOpenstackCommand(command)
	if err != nil {
		return nil, errors.Wrapf(err, "failed run 'openstack server create %s'", name)
	}
	result := NovaShow{}
	err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 30*time.Minute, true, func(_ context.Context) (done bool, err error) {
		stdout, _, err = c.RunOpenstackCommand(fmt.Sprintf("openstack server show %s -f json", name))
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed run 'openstack server show %s'", name)
			return false, nil
		}
		result = NovaShow{}
		err = json.Unmarshal([]byte(stdout), &result)
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to parse output for 'openstack server show %s'", name)
			return false, nil
		}
		TF.Log.Info().Msgf("Nova server current state:\n"+
			"Status: actual=%s, expected=ACTIVE, condition=%v;\n"+
			"Power: actual=%v, expected=1, condition=%v;\n"+
			"Keypair: actual=%s, expected=%s, condition=%v;\n"+
			"Image: actual=%s, expected=%s, condition=%v;\n"+
			"Flavor: actual=%s, expected=%s, condition=%v",
			result.Status, result.Status == "ACTIVE", result.Power, result.Power == 1,
			result.Keypair, keypair, result.Keypair == keypair,
			result.Image, image, strings.Contains(result.Image, image),
			result.Flavor, flavor, strings.Contains(result.Flavor, flavor),
		)
		if !waiting {
			return true, nil
		}
		return result.Status == "ACTIVE" &&
			result.Power == 1 &&
			result.Keypair == keypair &&
			strings.Contains(result.Image, image) &&
			strings.Contains(result.Flavor, flavor), nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to wait for 'openstack server create %s' completion", name)
	}
	return &result, nil
}

func (c *ManagedConfig) KeypairCreate(name string, privateFile string) (string, error) {
	command := fmt.Sprintf("openstack keypair create %s --private-key %s", name, privateFile)
	_, _, err := c.RunOpenstackCommand(command)
	if err != nil {
		return "", errors.Wrapf(err, "failed to run '%s'", command)
	}
	privateKey, _, err := c.RunOpenstackCommand(fmt.Sprintf("cat %s", privateFile))
	if err != nil {
		return "", errors.Wrap(err, "failed to read keypair private key from file")
	}
	return privateKey, nil
}

func (c *ManagedConfig) KeypairDelete(name string, privateFile string) error {
	command := fmt.Sprintf("openstack keypair delete %s", name)
	_, _, err := c.RunOpenstackCommand(command)
	if err != nil {
		return errors.Wrapf(err, "failed to run '%s'", command)
	}

	// remove private key file from keystone-client pod
	_, _, err = c.RunOpenstackCommand(fmt.Sprintf("rm %s", privateFile))
	if err != nil {
		TF.Log.Error().Err(err).Msgf("failed to remove %s file", privateFile)
	}
	return nil
}

func (c *ManagedConfig) NovaServerAddVolume(serverName, volumeName string) (string, error) {
	_, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server add volume %s %s", serverName, volumeName))
	if err != nil {
		return "", errors.Wrapf(err, "failed run 'openstack server add volume %s %s", serverName, volumeName)
	}

	device := ""
	err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 5*time.Minute, true, func(_ context.Context) (done bool, err error) {
		stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server show %s -f json", serverName))
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed run 'openstack server show %s'", serverName)
			return false, nil
		}
		server := NovaShow{}
		err = json.Unmarshal([]byte(stdout), &server)
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to parse output for 'openstack server show %s'", serverName)
			return false, nil
		}

		stdout, _, err = c.RunOpenstackCommand(fmt.Sprintf("openstack volume show %s -f json", volumeName))
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed run 'openstack volume show %s'", volumeName)
			return false, nil
		}
		volume := CinderShow{}
		err = json.Unmarshal([]byte(stdout), &volume)
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to parse output for 'openstack volume show %s'", volumeName)
			return false, nil
		}

		found := false
		for _, srvVol := range server.Volumes {
			if srvVol.ID == volume.ID {
				found = true
				break
			}
		}
		if !found {
			TF.Log.Info().Msgf("volume %s is not found in server %s volumes", volumeName, serverName)
			return false, nil
		}

		for _, volAtt := range volume.Attachments {
			if volAtt.ServerID == server.ID && volAtt.VolumeID == volume.ID {
				device = volAtt.Device
				return true, nil
			}
		}
		TF.Log.Info().Msgf("server %s is not found in volume %s attachments", serverName, volumeName)
		return false, nil
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to wait for volume %s attached to server %s", volumeName, serverName)
	}
	return device, nil
}

func (c *ManagedConfig) NovaServerRemoveVolume(serverName, volumeName string) error {
	_, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server remove volume %s %s", serverName, volumeName))
	if err != nil {
		return errors.Wrapf(err, "failed run 'openstack server remove volume %s %s", serverName, volumeName)
	}
	err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 5*time.Minute, true, func(_ context.Context) (done bool, err error) {
		stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server show %s -f json", serverName))
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed run 'openstack server show %s'", serverName)
			return false, nil
		}
		server := NovaShow{}
		err = json.Unmarshal([]byte(stdout), &server)
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to parse output for 'openstack server show %s'", serverName)
			return false, nil
		}

		stdout, _, err = c.RunOpenstackCommand(fmt.Sprintf("openstack volume show %s -f json", volumeName))
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed run 'openstack volume show %s'", volumeName)
			return false, nil
		}
		volume := CinderShow{}
		err = json.Unmarshal([]byte(stdout), &volume)
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to parse output for 'openstack volume show %s'", volumeName)
			return false, nil
		}

		for _, srvVol := range server.Volumes {
			if srvVol.ID == volume.ID {
				TF.Log.Info().Msgf("volume %s is still in server %s volumes", volumeName, serverName)
				return false, nil
			}
		}

		for _, volAtt := range volume.Attachments {
			if volAtt.ServerID == server.ID && volAtt.VolumeID == volume.ID {
				TF.Log.Info().Msgf("server %s is still in volume %s attachments", serverName, volumeName)
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to wait for volume %s detached from server %s", volumeName, serverName)
	}
	return nil
}

func (c *ManagedConfig) NovaServerImageCreate(serverName, imageName string, waiting bool) (*GlanceShow, error) {
	_, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server image create %s --name %s", serverName, imageName))
	if err != nil {
		return nil, errors.Wrapf(err, "failed run 'openstack server image create %s --name %s", serverName, imageName)
	}

	result := GlanceShow{}
	err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 30*time.Minute, true, func(_ context.Context) (done bool, err error) {
		stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack image show %s -f json", imageName))
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed run 'openstack image show %s'", imageName)
			return false, nil
		}
		result = GlanceShow{}
		err = json.Unmarshal([]byte(stdout), &result)
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to parse output for 'openstack image show %s'", imageName)
			return false, nil
		}
		if !waiting {
			return true, nil
		}
		return result.Status == "active", nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to wait for 'openstack server image create %s --name %s' completion", serverName, imageName)
	}
	return &result, nil
}

func (c *ManagedConfig) NovaServerReboot(name string, hard bool, waiting bool) (*NovaShow, error) {
	// create server
	timeBeforeReboot := time.Now()
	TF.Log.Info().Msgf("Reboot action initiated at %v", timeBeforeReboot)
	command := fmt.Sprintf("openstack server reboot %s", name)
	if hard {
		command += " --hard"
	} else {
		command += "  --soft"
	}
	_, _, err := c.RunOpenstackCommand(command)
	if err != nil {
		return nil, errors.Wrapf(err, "failed run 'openstack server reboot %s'", name)
	}
	result := NovaShow{}

	TF.Log.Info().Msgf("searching for server %s reboot event", name)
	err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 30*time.Minute, true, func(_ context.Context) (done bool, err error) {
		stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server event list %s -f json", name))
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed run 'openstack server event list %s -f json'", name)
			return false, nil
		}
		eventList := []NovaEventListItem{}
		err = json.Unmarshal([]byte(stdout), &eventList)
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to parse output for 'openstack server event list %s -f json'", name)
			return false, nil
		}
		// it should be just the first event from event list due to desc sort by start time
		eventTime, err := time.Parse(time.RFC3339, eventList[0].StartTime+"Z")
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to parse start time from event %s", eventList[0].ID)
		}
		if eventList[0].Action != "reboot" && timeBeforeReboot.Before(eventTime) {
			TF.Log.Info().Msgf("No reboot event yet for %s", name)
			return false, nil
		}
		TF.Log.Info().Msgf("Reboot event %s found started on %v", eventList[0].ID, eventTime)

		stdout, _, err = c.RunOpenstackCommand(fmt.Sprintf("openstack server event show %s %s -f json", name, eventList[0].ID))
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed run 'openstack server event show %s %s -f json'", name, eventList[0].ID)
			return false, nil
		}
		eventShow := NovaEventShow{}
		err = json.Unmarshal([]byte(stdout), &eventShow)
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to parse output for 'openstack server event show %s %s -f json'", name, eventList[0].ID)
			return false, nil
		}
		for _, event := range eventShow.Events {
			if event.Event == "compute_reboot_instance" {
				TF.Log.Info().Msgf("Current event reboot result is %s", event.Result)
				return event.Result == "Success", nil
			}
		}
		return false, nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to wait for 'openstack server reboot %s' event", name)
	}

	TF.Log.Info().Msgf("waiting for server %s become active", name)
	err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 30*time.Minute, true, func(_ context.Context) (done bool, err error) {
		stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server show %s -f json", name))
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed run 'openstack server show %s'", name)
			return false, nil
		}
		result = NovaShow{}
		err = json.Unmarshal([]byte(stdout), &result)
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to parse output for 'openstack server show %s'", name)
			return false, nil
		}
		if !waiting {
			return true, nil
		}
		return result.Status == "ACTIVE" && result.Power == 1, nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to wait for 'openstack server reboot %s' completion", name)
	}
	return &result, nil
}

func (c *ManagedConfig) SwiftContainerCreate(name string) error {
	_, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack container create %s", name))
	if err != nil {
		return errors.Wrapf(err, "failed to run 'openstack container create %s'", name)
	}
	return nil
}

func (c *ManagedConfig) SwiftObjectUpload(containerName, objectName, testString string) error {
	filePath := fmt.Sprintf("/tmp/testfile-%d", time.Now().Unix())
	_, _, err := c.RunOpenstackCommand(fmt.Sprintf(`bash -c "echo '%s' > %s"`, testString, filePath))
	if err != nil {
		return errors.Wrapf(err, "failed to run 'bash -c \"echo '%s' > %s'\"", testString, filePath)
	}
	stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("cat %s", filePath))
	if err != nil {
		return errors.Wrapf(err, "failed to run 'cat %s", testString)
	}
	_, _, err = c.RunOpenstackCommand(fmt.Sprintf("openstack object create --name %s %s %s", objectName, containerName, filePath))
	if err != nil {
		return errors.Wrapf(err, "STDOUT: %v, failed to run 'openstack object create --name %s %s %s'", stdout, objectName, containerName, filePath)
	}
	_, _, err = c.RunOpenstackCommand(fmt.Sprintf("rm %s", filePath))
	if err != nil {
		return errors.Wrapf(err, "failed to run 'rm %s'", filePath)
	}
	return nil
}

func (c *ManagedConfig) SwiftObjectDownload(containerName, objectName string) (string, error) {
	filePath := fmt.Sprintf("/tmp/testresult-%d", time.Now().Unix())
	_, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack object save --file %s %s %s", filePath, containerName, objectName))
	if err != nil {
		return "", errors.Wrapf(err, "failed to run 'openstack object create --file %s %s %s'", filePath, containerName, objectName)
	}
	content, _, err := c.RunOpenstackCommand(fmt.Sprintf("cat %s", filePath))
	if err != nil {
		return "", errors.Wrapf(err, "failed to run 'cat %s'", filePath)
	}
	_, _, err = c.RunOpenstackCommand(fmt.Sprintf("rm %s", filePath))
	if err != nil {
		return "", errors.Wrapf(err, "failed to run 'rm %s'", filePath)
	}
	return content, err
}

func (c *ManagedConfig) SwiftContainerDelete(name string) error {
	stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack object list %s -f value", name))
	if err != nil {
		return errors.Wrapf(err, "failed to run 'openstack object list %s -f value'", name)
	}
	objects := strings.Split(stdout, "\n")

	removed := true
	for _, obj := range objects {
		if obj != "" {
			_, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack object delete %s %s", name, obj))
			if err != nil {
				TF.Log.Error().Err(err).Msgf("failed to run 'openstack object delete %s %s'", name, obj)
				removed = false
			}
		}
	}
	if !removed {
		return errors.Errorf("failed to remove some objects from container %s", name)
	}

	err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 5*time.Minute, true, func(_ context.Context) (done bool, err error) {
		_, _, err = c.RunOpenstackCommand(fmt.Sprintf("openstack container delete %s", name))
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to run 'openstack container delete %s'", name)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to remove swift container %s", name)
	}

	return nil
}

func (c *ManagedConfig) NovaServerDisablePortSecurity(serverName string) error {
	_, _, err := c.RunOpenstackCommand("openstack net set --external public")
	if err != nil {
		return errors.Wrapf(err, "failed to run 'openstack net set --external public'")
	}
	_, _, err = c.RunOpenstackCommand("openstack subnet set --dhcp public-subnet")
	if err != nil {
		return errors.Wrapf(err, "failed to run 'openstack subnet set --dhcp public-subnet'")
	}

	//Getting server
	stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server show %s -f json", serverName))
	if err != nil {
		return errors.Wrapf(err, "failed to run 'openstack server show %s'", serverName)
	}
	server := NovaShow{}
	err = json.Unmarshal([]byte(stdout), &server)
	if err != nil {
		return errors.Wrapf(err, "failed to parse output for 'openstack server show %s'", serverName)
	}

	//Parsing its IP address
	var serverIP string
	for addr := range server.Addresses {
		serverIP = server.Addresses[addr][0]
		break
	}

	//List all ports
	stdout, _, err = c.RunOpenstackCommand("openstack port list -f json")
	if err != nil {
		return errors.Wrap(err, "failed to run 'openstack port list'")
	}
	ports := []PortListItem{}
	err = json.Unmarshal([]byte(stdout), &ports)
	if err != nil {
		return errors.Wrap(err, "failed to parse output for 'openstack port list'")
	}

	portID := ""
	for _, port := range ports {
		for _, fixedIP := range port.FixedIPs {
			if fixedIP.IP == serverIP {
				portID = port.ID
				break
			}
		}
	}
	if portID == "" {
		return errors.Errorf("failed to find port with IP address %s", serverIP)
	}

	_, stderr, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server remove security group %s default", serverName))
	if err != nil && !strings.Contains(stderr, "not associated with the instance") {
		return errors.Wrapf(err, "failed to run 'openstack server remove security group %s default'", serverName)
	}

	_, _, err = c.RunOpenstackCommand(fmt.Sprintf("openstack port set %s --disable-port-security", portID))
	if err != nil {
		return errors.Wrapf(err, "failed to run 'openstack port set %s --disable-port-security'", portID)
	}

	return nil
}

func (c *ManagedConfig) NovaServerMigrate(name string, waiting bool) (*NovaShow, error) {
	_, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server migrate %s", name))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run 'openstack server migrate %s'", name)
	}
	result := NovaShow{}
	err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 30*time.Minute, true, func(_ context.Context) (done bool, err error) {
		stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server show %s -f json", name))
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed run 'openstack server show %s'", name)
			return false, nil
		}
		result = NovaShow{}
		err = json.Unmarshal([]byte(stdout), &result)
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to parse output for 'openstack server show %s'", name)
			return false, nil
		}
		if !waiting {
			return true, nil
		}
		return result.Status == "VERIFY_RESIZE", nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to wait for 'openstack server migrate %s' completion", name)
	}
	return &result, nil
}

func (c *ManagedConfig) NovaServerMigrateAction(name string, action string, waiting bool) (*NovaShow, error) {
	if action != "confirm" && action != "revert" {
		return nil, errors.Errorf("incorrect '%s' migrate action, allowed are: confirm, revert", action)
	}
	_, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server migrate %s %s", action, name))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run 'openstack server migrate %s %s'", action, name)
	}
	result := NovaShow{}
	err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 30*time.Minute, true, func(_ context.Context) (done bool, err error) {
		stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack server show %s -f json", name))
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed run 'openstack server show %s'", name)
			return false, nil
		}
		result = NovaShow{}
		err = json.Unmarshal([]byte(stdout), &result)
		if err != nil {
			TF.Log.Error().Err(err).Msgf("failed to parse output for 'openstack server show %s'", name)
			return false, nil
		}
		if !waiting {
			return true, nil
		}
		return result.Status == "ACTIVE", nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to wait for 'openstack server migrate %s %s' completion", action, name)
	}
	return &result, nil
}

func (c *ManagedConfig) GetOpenstackDeployment() (*unstructured.Unstructured, error) {
	osdplResource := schema.GroupVersionResource{Group: "lcm.mirantis.com", Version: "v1alpha1", Resource: "openstackdeployments"}
	osdpl, err := c.DynamicClient.Resource(osdplResource).Namespace("openstack").Get(c.Context, "osh-dev", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get openstackdeployment openstack/osh-dev")
	}
	return osdpl, nil
}

func (c *ManagedConfig) GetOpenstackDeploymentStatusState() (string, error) {
	u := &unstructured.UnstructuredList{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "lcm.mirantis.com",
		Kind:    "OpenStackDeploymentStatusList",
		Version: "v1alpha1",
	})
	err := c.Client.List(c.Context, u, &client.ListOptions{Namespace: "openstack"})
	if client.IgnoreNotFound(err) != nil {
		if !meta.IsNoMatchError(err) && !runtime.IsNotRegisteredError(err) {
			return "", errors.Wrap(err, "failed to list openstackdeploymentstatus objects")
		}
		return "", errors.Wrap(err, "no openstackdeploymentstatus objects found")
	}
	if len(u.Items) == 0 {
		return "", nil
	}
	if len(u.Items) != 1 {
		return "", errors.Errorf("expected number of openstackdeploymentstatus objects is 1, but found %d", len(u.Items))
	}

	osdplst := u.Items[0]
	return osdplst.Object["status"].(map[string]interface{})["osdpl"].(map[string]interface{})["state"].(string), nil
}

func (c *ManagedConfig) UpdateOpenstackDeployment(data map[string]interface{}, waitApplied bool) error {
	TF.Log.Info().Msg("Updating openstackdeployment")
	osdpl, err := c.GetOpenstackDeployment()
	if err != nil {
		return errors.Wrap(err, "failed to get OpenstackDeployment")
	}
	osdpl.Object["spec"] = data["spec"]
	osdplResource := schema.GroupVersionResource{Group: "lcm.mirantis.com", Version: "v1alpha1", Resource: "openstackdeployments"}
	_, err = c.DynamicClient.Resource(osdplResource).Namespace("openstack").Update(c.Context, osdpl, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update OpenstackDeployment")
	}
	if waitApplied {
		TF.Log.Info().Msg("Waiting for OpenStackDeployment status to be APPLIED")
		err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 15*time.Minute, true, func(_ context.Context) (done bool, err error) {
			state, err := c.GetOpenstackDeploymentStatusState()
			if err != nil {
				TF.Log.Error().Err(err).Msg("failed to get osdplst state")
				return false, nil
			}
			TF.Log.Info().Msgf("OpenStackDeployment status state wait for APPLYING: %s", state)
			return state == "APPLYING", nil
		})
		if err != nil {
			return errors.Wrap(err, "Failed to wait for osdplst state become APPLYING")
		}

		err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 15*time.Minute, true, func(_ context.Context) (done bool, err error) {
			state, err := c.GetOpenstackDeploymentStatusState()
			if err != nil {
				TF.Log.Error().Err(err).Msg("failed to get osdplst state")
				return false, nil
			}
			TF.Log.Info().Msgf("OpenStackDeployment status state wait for APPLIED: %s", state)
			return state == "APPLIED", nil
		})
		if err != nil {
			return errors.Wrap(err, "Failed to wait for osdplst state become APPLIED")
		}
	}
	return nil
}

func BuildManilaCephFSDriver(osdpl *unstructured.Unstructured, params *ManilaConfigParams) (map[string]interface{}, error) {
	data := osdpl.Object
	if data["spec"] == nil {
		return nil, errors.New("osdpl object spec not found")
	}

	if data["spec"].(map[string]interface{})["services"] == nil {
		data["spec"].(map[string]interface{})["services"] = map[string]interface{}{}
	}
	data["spec"].(map[string]interface{})["services"].(map[string]interface{})["shared-file-system"] = map[string]interface{}{
		"manila": map[string]interface{}{
			"values": map[string]interface{}{
				"conf": map[string]interface{}{
					"ceph": map[string]interface{}{
						"config": map[string]interface{}{
							fmt.Sprintf("client.%s", params.CephFsClientName): map[string]interface{}{
								"client mount gid": "0",
								"client mount uid": "0",
								"keyring":          fmt.Sprintf("/etc/ceph/ceph.client.%s.keyring", params.CephFsClientName),
							},
							"global": map[string]interface{}{
								"mon_host": strings.Split(params.MonEndpoints, ","),
							},
						},
						"keyrings": map[string]interface{}{
							params.CephFsClientName: map[string]interface{}{
								"key": params.CephFsClientKey,
							},
						},
					},
					"manila": map[string]interface{}{
						"DEFAULT": map[string]interface{}{
							"default_share_type":      "cephfs",
							"enabled_share_backends":  "cephfs",
							"enabled_share_protocols": "CEPHFS",
						},
					},
					"standalone_backends": map[string]interface{}{
						"statefulsets": map[string]interface{}{
							"cephfs": map[string]interface{}{
								"conf": map[string]interface{}{
									"DEFAULT": map[string]interface{}{
										"enabled_share_backends":  "cephfs",
										"enabled_share_protocols": "CEPHFS",
									},
									"cephfs": map[string]interface{}{
										"cephfs_auth_id":               params.CephFsClientName,
										"cephfs_cluster_name":          "ceph",
										"cephfs_conf_path":             "/etc/ceph/ceph.conf",
										"cephfs_filesystem_name":       params.CephFsName,
										"cephfs_protocol_helper_type":  "CEPHFS",
										"driver_handles_share_servers": false,
										"share_backend_name":           "cephfs",
										"share_driver":                 "manila.share.drivers.cephfs.driver.CephFSDriver",
									},
								},
							},
						},
					},
				},
				"manifests": map[string]interface{}{
					"ceph_conf":         true,
					"daemonset_share":   false,
					"statefulset_share": true,
				},
			},
		},
	}
	osdpl.Object = data
	return data, nil
}

func (c *ManagedConfig) CreateManilaShareType() error {
	_, stderr, err := c.RunOpenstackCommand("openstack share type create cephfs true")
	if err != nil {
		errMsg := "failed to exec 'openstack share type create cephfs true'"
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %s)", stderr)
		}
		return errors.Wrap(err, errMsg)
	}

	_, stderr, err = c.RunOpenstackCommand("openstack share type set cephfs --extra-specs driver_handles_share_servers=false vendor_name=Ceph storage_protocol=CEPHFS")
	if err != nil {
		errMsg := "failed to exec 'openstack share type set cephfs'"
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %s)", stderr)
		}
		return errors.Wrap(err, errMsg)
	}

	return nil
}

func (c *ManagedConfig) DeleteManilaShareType() error {
	_, stderr, err := c.RunOpenstackCommand("openstack share type delete cephfs")
	if err != nil {
		errMsg := "failed to exec 'openstack share type delete cephfs'"
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %s)", stderr)
		}
		return errors.Wrap(err, errMsg)
	}
	return nil
}

func (c *ManagedConfig) CreateManilaShare(name string, shareClientName string) (string, string, error) {
	cmd := fmt.Sprintf("openstack share create CEPHFS 1 --name %s --share-type cephfs", name)
	_, stderr, err := c.RunOpenstackCommand(cmd)
	if err != nil {
		errMsg := "failed to exec 'openstack share create'"
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %s)", stderr)
		}
		return "", "", errors.Wrap(err, errMsg)
	}

	_, stderr, err = c.RunOpenstackCommand(fmt.Sprintf("openstack share access create %s cephx %s", name, shareClientName))
	if err != nil {
		errMsg := "failed to exec 'openstack share access create'"
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %s)", stderr)
		}
		return "", "", errors.Wrap(err, errMsg)
	}

	// get location
	stdout, stderr, err := c.RunOpenstackCommand(fmt.Sprintf("openstack share export location list %s -f json", name))
	if err != nil {
		errMsg := "failed to exec 'openstack share export location list'"
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %s)", stderr)
		}
		return "", "", errors.Wrap(err, errMsg)
	}
	result := []ManilaExportLocationItem{}
	err = json.Unmarshal([]byte(stdout), &result)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to parse output for 'openstack share export location list'")
	}
	location := strings.SplitN(result[0].Path, "/", 2)[1]

	// get shareClient keyring
	stdout, stderr, err = c.RunOpenstackCommand(fmt.Sprintf("openstack share access list %s -f json", name))
	if err != nil {
		errMsg := "failed to exec 'openstack share access list'"
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %s)", stderr)
		}
		return "", "", errors.Wrap(err, errMsg)
	}
	accessList := []ManilaShareAccessItem{}
	err = json.Unmarshal([]byte(stdout), &accessList)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to parse output for 'openstack share access list'")
	}
	accessKey := ""
	for _, access := range accessList {
		if access.AccessTo == shareClientName {
			accessKey = access.AccessKey
			break
		}
	}
	if accessKey == "" {
		return "", "", errors.New("failed to find Access Key for share access")
	}
	return "/" + location, accessKey, nil
}

func (c *ManagedConfig) DeleteManilaShare(name string) error {
	cmd := fmt.Sprintf("openstack share delete %s", name)
	_, stderr, err := c.RunOpenstackCommand(cmd)
	if err != nil {
		errMsg := "failed to exec 'openstack share delete'"
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %s)", stderr)
		}
		return errors.Wrap(err, errMsg)
	}
	return nil
}

func (c *ManagedConfig) HeatStackShow(name string) (*StackShow, error) {
	stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack stack show %s -f json", name))
	if err != nil {
		return nil, errors.Wrapf(err, "failed run 'openstack stack show %s'", name)
	}
	result := StackShow{}
	err = json.Unmarshal([]byte(stdout), &result)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse output for 'openstack stack show %s'", name)
	}
	return &result, nil
}

func (c *ManagedConfig) HeatStackOutputShow(stackName, outputKey string) (*StackOutputShow, error) {
	stdout, _, err := c.RunOpenstackCommand(fmt.Sprintf("openstack stack output show %s %s -f json", stackName, outputKey))
	if err != nil {
		return nil, errors.Wrapf(err, "failed run 'openstack stack output show %s %s'", stackName, outputKey)
	}
	result := StackOutputShow{}
	err = json.Unmarshal([]byte(stdout), &result)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse output for 'openstack stack output show %s %s'", stackName, outputKey)
	}
	return &result, nil
}

func (c *ManagedConfig) HeatStackCreate(name, defaultsPath, templatePath string) error {
	_, stderr, err := c.RunOpenstackCommand(fmt.Sprintf("openstack stack create -t %s -e %s %s", templatePath, defaultsPath, name))
	if err != nil {
		errMsg := fmt.Sprintf("failed to exec 'openstack stack create -t %s -e %s %s'", templatePath, defaultsPath, name)
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %s)", stderr)
		}
		return errors.Wrap(err, errMsg)
	}
	err = wait.PollUntilContextTimeout(c.Context, 15*time.Second, 15*time.Minute, true, func(_ context.Context) (bool, error) {
		stack, err := c.HeatStackShow(name)
		if err != nil {
			TF.Log.Error().Err(err).Msg("failed to show heat stack")
			return false, nil
		}
		if stack.Status == "CREATE_FAILED" {
			return false, errors.New("stack status is CREATE_FAILED")
		}
		return stack.Status == "CREATE_COMPLETE", nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to wait for CREATE_COMPLETE status of stack")
	}
	return nil
}

func (c *ManagedConfig) HeatStackDelete(name string) error {
	_, stderr, err := c.RunOpenstackCommand(fmt.Sprintf("openstack stack delete %s", name))
	if err != nil {
		errMsg := fmt.Sprintf("failed to exec 'openstack stack delete %s'", name)
		if stderr != "" {
			errMsg += fmt.Sprintf(" (stderr: %s)", stderr)
		}
		return errors.Wrap(err, errMsg)
	}
	return nil
}
