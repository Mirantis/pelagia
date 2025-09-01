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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/pkg/errors"
	rookv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
)

type StoreState struct {
	CephDeployment *cephlcmv1alpha1.CephDeployment
	CephCluster    *rookv1.CephCluster
	ExportDir      string
}

func NewStoreState() (*StoreState, error) {
	storeState := &StoreState{}
	cephdpl, err := TF.ManagedCluster.FindCephDeployment()
	if err != nil {
		return nil, err
	}
	storeState.CephDeployment = cephdpl
	cephcluster, err := TF.ManagedCluster.GetCephCluster(cephdpl.Name)
	if err != nil {
		return nil, err
	}
	storeState.CephCluster = cephcluster
	storeState.ExportDir = os.Getenv("EXPORT_DIR")
	return storeState, nil
}

func (s *StoreState) CopyState(t *StoreState) {
	if s.CephDeployment != nil {
		t.CephDeployment = s.CephDeployment.DeepCopy()
	}
	if s.CephCluster != nil {
		t.CephCluster = s.CephCluster.DeepCopy()
	}
	t.ExportDir = s.ExportDir
}

func (s *StoreState) RestoreStoredState() error {
	TF.Log.Info().Msg("waiting for CephDeployment restored...")
	if s.CephDeployment == nil {
		TF.Log.Info().Msg("skipping CephDeployment restore since no previous version (looks like created in tests)")
		return nil
	}
	restoreErr := wait.PollUntilContextTimeout(TF.ManagedCluster.Context, 5*time.Second, 3*time.Minute, true, func(_ context.Context) (bool, error) {
		curCephDpl, clusterPresentErr := TF.ManagedCluster.GetCephDeployment(s.CephDeployment.Name)
		if clusterPresentErr != nil {
			if !apierrors.IsNotFound(clusterPresentErr) {
				TF.Log.Error().Err(clusterPresentErr).Msg("")
				return false, clusterPresentErr
			}
			err := TF.ManagedCluster.CreateCephDeployment(s.CephDeployment)
			if err != nil {
				TF.Log.Error().Err(err).Msg("")
				return false, err
			}
		} else if !reflect.DeepEqual(curCephDpl.Spec, s.CephDeployment.Spec) {
			lcmcommon.ShowObjectDiff(TF.Log, curCephDpl.Spec, s.CephDeployment.Spec)
			curCephDpl.Spec = s.CephDeployment.Spec
			_, err := TF.ManagedCluster.UpdateCephDeploymentSpec(curCephDpl)
			if err != nil {
				TF.Log.Error().Err(err).Msg("")
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
	if restoreErr != nil {
		return errors.Wrap(restoreErr, "unable to restore stored CephDeployment state")
	}
	err := TF.ManagedCluster.WaitForCephDeploymentReady(s.CephDeployment.Name)
	if err != nil {
		return errors.Wrap(err, "failed to wait for CephDeployment readiness")
	}
	return nil
}

func (s *StoreState) ConvertStateToJSON() (state []byte, err error) {
	var out bytes.Buffer
	s.CephCluster.Kind = "CephCluster"
	s.CephCluster.APIVersion = "ceph.rook.io/v1"
	outputJSON := []interface{}{s.CephCluster}
	s.CephDeployment.Kind = "CephDeployment"
	s.CephDeployment.APIVersion = "lcm.mirantis.com/v1alpha1"
	outputJSON = append(outputJSON, s.CephDeployment)
	jsonData, err := json.Marshal(outputJSON)
	if err != nil {
		err = errors.Wrap(err, "failed to convert State objects to json data")
		return
	}
	err = json.Indent(&out, jsonData, "", "  ")
	if err != nil {
		err = errors.Wrap(err, "failed to convert State objects to pretty json data")
		return
	}
	state = out.Bytes()
	return
}

func (s *StoreState) ExportStoreState(dirname string) error {
	if _, err := os.Stat(dirname); os.IsNotExist(err) {
		err := os.Mkdir(dirname, 0777)
		if err != nil {
			return errors.Wrapf(err, "failed to create export directory %s", dirname)
		}
	}

	fileName := fmt.Sprintf("%s/export-failed-%v.json", dirname, time.Now().Unix())
	newExportFile, err := os.Create(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to create new export file %s for failed tests", fileName)
	}

	stateJSON, err := s.ConvertStateToJSON()
	if err != nil {
		return err
	}
	_, err = newExportFile.Write(stateJSON)
	if err != nil {
		return errors.Wrapf(err, "failed to write to file %s for failed tests", fileName)
	}
	return nil
}
