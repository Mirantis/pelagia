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

package input

import (
	"fmt"

	"sigs.k8s.io/yaml"

	cephlcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

func ConvertJSONToYaml(data []byte) []byte {
	data, err := yaml.JSONToYAML(data)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	return data
}

func ConvertYamlToJSON(data []byte) []byte {
	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	return jsonData
}

func ConvertStructToRaw(s any) []byte {
	data, err := cephlcmv1alpha1.DecodeStructToRaw(s)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	return data
}
