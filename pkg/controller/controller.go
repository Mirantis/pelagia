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

package controller

import (
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	lcmdeployment "github.com/Mirantis/pelagia/pkg/controller/deployment"
	lcmhealth "github.com/Mirantis/pelagia/pkg/controller/health"
	lcminfra "github.com/Mirantis/pelagia/pkg/controller/infra"
	lcmosdremove "github.com/Mirantis/pelagia/pkg/controller/osdremove-task"
	lcmsecret "github.com/Mirantis/pelagia/pkg/controller/secret"
)

// AddToManagerFuncs is a map of functions to add all Controllers to the Manager
var AddToManagerFuncs = map[string]func(manager.Manager) error{}

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, controllerName string) error {
	if _, ok := AddToManagerFuncs[controllerName]; !ok {
		return errors.Errorf("Failed to find reconciler for '%s' controller", controllerName)
	}
	return AddToManagerFuncs[controllerName](m)
}

// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
func init() {
	AddToManagerFuncs[lcmdeployment.ControllerName] = lcmdeployment.Add
	AddToManagerFuncs[lcmhealth.ControllerName] = lcmhealth.Add
	AddToManagerFuncs[lcminfra.ControllerName] = lcminfra.Add
	AddToManagerFuncs[lcmosdremove.ControllerName] = lcmosdremove.Add
	AddToManagerFuncs[lcmsecret.ControllerName] = lcmsecret.Add
}
