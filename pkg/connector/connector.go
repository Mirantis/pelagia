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

package connector

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"
	rookclient "github.com/rook/rook/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

type CephConnector struct {
	Context       context.Context
	Config        *rest.Config
	Kubeclientset kubernetes.Interface
	Rookclientset rookclient.Interface
}

type Opts struct {
	RookNamespace    string
	AuthClient       string
	UseRBD           bool
	UseCephFS        bool
	UseRgw           bool
	RgwUserName      string
	ToolBoxLabel     string
	ToolBoxNamespace string
	EncodedBase64    bool
}

func GetConnector() (*CephConnector, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client config")
	}
	rookClientset, err := rookclient.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rook client")
	}
	kubeClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubernetes client")
	}
	return &CephConnector{Config: config, Kubeclientset: kubeClientset, Rookclientset: rookClientset, Context: context.TODO()}, nil
}

func (c *CephConnector) PrepareConnectionString(opts Opts) (string, error) {
	connectionInfo, err := c.getConnectionInfo(opts)
	if err != nil {
		return "", errors.Wrapf(err, "failed to prepare connection info")
	}
	s, _ := json.Marshal(connectionInfo)
	if opts.EncodedBase64 {
		return base64.StdEncoding.EncodeToString([]byte(s)), nil
	}
	return string(s), nil
}

func (c *CephConnector) getConnectionInfo(opts Opts) (*lcmcommon.CephConnection, error) {
	newConnectionInfo := &lcmcommon.CephConnection{ClientName: opts.AuthClient}
	fsid, err := c.getClusterFSID(opts.RookNamespace)
	if err != nil {
		return nil, err
	}

	monEndpoints, err := c.getClusterMonEndpoints(opts.RookNamespace)
	if err != nil {
		return nil, err
	}

	clientKeyring, err := c.getClusterClientKeyring(opts.ToolBoxNamespace, opts.ToolBoxLabel, opts.AuthClient)
	if err != nil {
		return nil, err
	}
	newConnectionInfo.FSID = fsid
	newConnectionInfo.MonEndpoints = monEndpoints
	newConnectionInfo.ClientKeyring = clientKeyring

	if opts.AuthClient != "admin" {
		if opts.UseRBD {
			nodeKeyring, err := c.getClusterClientKeyring(opts.ToolBoxNamespace, opts.ToolBoxLabel, lcmcommon.CephCSIRBDNodeClientName)
			if err != nil {
				return nil, err
			}
			provisionerKeyring, err := c.getClusterClientKeyring(opts.ToolBoxNamespace, opts.ToolBoxLabel, lcmcommon.CephCSIRBDProvisionerClientName)
			if err != nil {
				return nil, err
			}
			newConnectionInfo.RBDKeyring = &lcmcommon.CSIKeyring{
				NodeKey:        nodeKeyring,
				ProvisionerKey: provisionerKeyring,
			}
		}
		if opts.UseCephFS {
			nodeKeyring, err := c.getClusterClientKeyring(opts.ToolBoxNamespace, opts.ToolBoxLabel, lcmcommon.CephCSICephFSNodeClientName)
			if err != nil {
				return nil, err
			}
			provisionerKeyring, err := c.getClusterClientKeyring(opts.ToolBoxNamespace, opts.ToolBoxLabel, lcmcommon.CephCSICephFSProvisionerClientName)
			if err != nil {
				return nil, err
			}
			newConnectionInfo.CephFSKeyring = &lcmcommon.CSIKeyring{
				NodeKey:        nodeKeyring,
				ProvisionerKey: provisionerKeyring,
			}
		}
	}

	if opts.UseRgw {
		rgwAdminOpsUserKeys, err := c.getRgwKeys(opts.ToolBoxNamespace, opts.ToolBoxLabel, opts.RgwUserName)
		if err != nil {
			return nil, err
		}
		newConnectionInfo.RgwAdminUserKeys = rgwAdminOpsUserKeys
	}

	return newConnectionInfo, nil
}
