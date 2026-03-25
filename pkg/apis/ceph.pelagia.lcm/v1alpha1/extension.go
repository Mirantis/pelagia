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

package v1alpha1

import (
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
)

func (cl *CephCluster) GetSpec() (cephv1.ClusterSpec, error) {
	var cephClusterSpec cephv1.ClusterSpec
	if cl == nil {
		return cephClusterSpec, errors.New("spec: cluster field has nil pointer value provided")
	}
	if cl.Raw == nil && cl.Object == nil {
		return cephClusterSpec, errors.New("spec: cluster field has no any data provided")
	}

	if cl.Raw != nil {
		if err := DecodeRawToStruct(cl.Raw, &cephClusterSpec); err != nil {
			return cephClusterSpec, errors.Wrap(err, "spec: cluster field has failed to decode to Rook ClusterSpec struct")
		}
		return cephClusterSpec, nil
	}

	cluster, ok := cl.Object.(*cephv1.CephCluster)
	if !ok {
		return cephClusterSpec, errors.New("spec: cluster field has failed to convert to Rook CephCluster object")
	}
	return cluster.Spec, nil
}

func (p CephPool) GetSpec() (cephv1.PoolSpec, error) {
	var cephPool cephv1.PoolSpec
	if p.PoolSpec.Raw == nil && p.PoolSpec.Object == nil {
		return cephPool, errors.New("spec: pool spec no any data provided")
	}

	if p.PoolSpec.Raw != nil {
		if err := DecodeRawToStruct(p.PoolSpec.Raw, &cephPool); err != nil {
			return cephPool, errors.Wrap(err, "spec: pool spec has failed to decode to Rook PoolSpec struct")
		}
		return cephPool, nil
	}

	pool, ok := p.PoolSpec.Object.(*cephv1.CephBlockPool)
	if !ok {
		return cephPool, errors.New("spec: pool field has failed to convert to Rook CephBlockPool object")
	}
	return pool.Spec.PoolSpec, nil
}

func (c CephClient) GetSpec() (cephv1.ClientSpec, error) {
	var clientSpec cephv1.ClientSpec
	if c.Raw == nil && c.Object == nil {
		return clientSpec, errors.New("spec: client spec no any data provided")
	}

	if c.Raw != nil {
		if err := DecodeRawToStruct(c.Raw, &clientSpec); err != nil {
			return clientSpec, errors.Wrap(err, "spec: client spec has failed to decode to Rook ClientSpec struct")
		}
		return clientSpec, nil
	}

	client, ok := c.Object.(*cephv1.CephClient)
	if !ok {
		return clientSpec, errors.New("spec: client field has failed to convert to Rook CephClient object")
	}
	return client.Spec, nil
}

func (fs CephFilesystem) GetSpec() (cephv1.FilesystemSpec, error) {
	var fsSpec cephv1.FilesystemSpec
	if fs.FsSpec.Raw == nil && fs.FsSpec.Object == nil {
		return fsSpec, errors.New("spec: filesystem spec no any data provided")
	}

	if fs.FsSpec.Raw != nil {
		if err := DecodeRawToStruct(fs.FsSpec.Raw, &fsSpec); err != nil {
			return fsSpec, errors.Wrap(err, "spec: filesystem spec has failed to decode to Rook FilesystemSpec struct")
		}
		return fsSpec, nil
	}

	fsObj, ok := fs.FsSpec.Object.(*cephv1.CephFilesystem)
	if !ok {
		return fsSpec, errors.New("spec: filesystem field has failed to convert to Rook CephFilesystem object")
	}
	return fsObj.Spec, nil
}

// Method SetRawSpec is used to directly put Raw spec in related
// fields to avoid full struct define after JSON marshaling
// Caution: will override present Raw data fully!
func SetRawSpec(re *runtime.RawExtension, rawData []byte, t any) error {
	if re == nil {
		return errors.New("RawExtension is on nil pointer")
	}
	if t != nil {
		err := DecodeRawToStruct(rawData, t)
		if err != nil {
			return err
		}
	}

	re.Raw = rawData
	re.Object = nil
	return nil
}

// Method DecodeRawToStruct is used to decode Raw data to
// some provided known structure and check that decoding
// can be exectuted
func DecodeRawToStruct(rawData []byte, r any) error {
	dec := json.NewDecoder(bytes.NewReader(rawData))
	dec.DisallowUnknownFields()
	if err := dec.Decode(r); err != nil {
		return errors.Wrapf(err, "failed to decode RawExtension to %T object, expected type mismatch", r)
	}
	return nil
}

func DecodeStructToRaw(r any) ([]byte, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return data, nil
}
