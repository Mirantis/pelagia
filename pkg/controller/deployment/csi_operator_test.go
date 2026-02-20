/*
Copyright 2026 Mirantis IT.

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

package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"

	csiopapi "github.com/ceph/ceph-csi-operator/api/v1"

	faketestclients "github.com/Mirantis/pelagia/test/unit/clients"
	unitinputs "github.com/Mirantis/pelagia/test/unit/inputs"
)

func TestDropCsiOperatorResources(t *testing.T) {
	builder := faketestclients.GetClientBuilder().WithLists(unitinputs.ClientProfilesRook.DeepCopy(),
		unitinputs.CsiDriversRook.DeepCopy(), unitinputs.CephConnectionsRook.DeepCopy(), unitinputs.OperatorConfigsRook.DeepCopy())
	c := fakeDeploymentConfig(nil, nil)
	c.api.ClientNoCache = faketestclients.GetClient(builder)

	tests := []struct {
		name    string
		present string
		removed bool
	}{
		{
			name: "removing clientprofile first",
		},
		{
			name: "removing other resources",
		},
		{
			name:    "no resources to remove",
			removed: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			removed, err := c.deleteCsiOperatorResources()
			assert.Nil(t, err)
			assert.Equal(t, test.removed, removed)
		})
	}
}

func TestDropCsiClientProfile(t *testing.T) {
	tests := []struct {
		name              string
		clientProfileList *csiopapi.ClientProfileList
		removed           bool
	}{
		{
			name:              "cephconnection is removing",
			clientProfileList: unitinputs.ClientProfilesRook.DeepCopy(),
		},
		{
			name:              "no cephconnection to remove",
			clientProfileList: unitinputs.ClientProfilesEmpty,
			removed:           true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			builder := faketestclients.GetClientBuilder()
			if test.clientProfileList != nil {
				builder = builder.WithLists(test.clientProfileList)
			}
			c.api.ClientNoCache = faketestclients.GetClient(builder)

			removed, err := c.deleteCsiClientProfile()
			assert.Nil(t, err)
			assert.Equal(t, test.removed, removed)
		})
	}
}

func TestDropCsiCephConnection(t *testing.T) {
	tests := []struct {
		name               string
		cephConnectionList *csiopapi.CephConnectionList
		removed            bool
	}{
		{
			name:               "cephconnection is removing",
			cephConnectionList: unitinputs.CephConnectionsRook.DeepCopy(),
		},
		{
			name:               "no cephconnection to remove",
			cephConnectionList: unitinputs.CephConnectionsEmpty,
			removed:            true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			builder := faketestclients.GetClientBuilder()
			if test.cephConnectionList != nil {
				builder = builder.WithLists(test.cephConnectionList)
			}
			c.api.ClientNoCache = faketestclients.GetClient(builder)

			removed, err := c.deleteCsiCephConnection()
			assert.Nil(t, err)
			assert.Equal(t, test.removed, removed)
		})
	}
}

func TestDropCsiOperatorConfig(t *testing.T) {
	tests := []struct {
		name         string
		opConfigList *csiopapi.OperatorConfigList
		removed      bool
	}{
		{
			name:         "operator config is removing",
			opConfigList: unitinputs.OperatorConfigsRook.DeepCopy(),
		},
		{
			name:         "some operator config is present, but not for our cluster",
			opConfigList: &csiopapi.OperatorConfigList{Items: []csiopapi.OperatorConfig{unitinputs.GetOperatorConfig("custom", "custom")}},
			removed:      true,
		},
		{
			name:         "no operator config to remove",
			opConfigList: unitinputs.OperatorConfigsEmpty,
			removed:      true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			builder := faketestclients.GetClientBuilder()
			if test.opConfigList != nil {
				builder = builder.WithLists(test.opConfigList)
			}
			c.api.ClientNoCache = faketestclients.GetClient(builder)

			removed, err := c.deleteCsiOperatorConfig()
			assert.Nil(t, err)
			assert.Equal(t, test.removed, removed)
		})
	}
}

func TestDropCsiDrivers(t *testing.T) {
	tests := []struct {
		name       string
		driverList *csiopapi.DriverList
		removed    bool
	}{
		{
			name:       "drivers are removing",
			driverList: unitinputs.CsiDriversRook.DeepCopy(),
		},
		{
			name:       "some drivers are present, but not for our cluster",
			driverList: &csiopapi.DriverList{Items: []csiopapi.Driver{unitinputs.GetCsiDriver("custom", "custom")}},
			removed:    true,
		},
		{
			name:       "no drivers to remove",
			driverList: unitinputs.CsiDriversEmpty,
			removed:    true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := fakeDeploymentConfig(nil, nil)
			builder := faketestclients.GetClientBuilder()
			if test.driverList != nil {
				builder = builder.WithLists(test.driverList)
			}
			c.api.ClientNoCache = faketestclients.GetClient(builder)

			removed, err := c.deleteCsiDrivers()
			assert.Nil(t, err)
			assert.Equal(t, test.removed, removed)
		})
	}
}
