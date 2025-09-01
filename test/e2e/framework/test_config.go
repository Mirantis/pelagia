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
	"fmt"
	"os"
	"testing"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type TestConfig struct {
	Cases    []string     `yaml:"testCases"`
	Settings TestSettings `yaml:"testSettings,omitempty"`
}

type CaseSettings struct {
	Name   string            `yaml:"name"`
	Config map[string]string `yaml:"config"`
}

type TestSettings struct {
	Namespace      string         `yaml:"namespace,omitempty"`
	CaseSettings   []CaseSettings `yaml:"caseSettings,omitempty"`
	KubeconfigURL  string         `yaml:"kubeconfigUrl,omitempty"`
	KubeconfigFile string         `yaml:"kubeconfigFile,omitempty"`
	KeepAfter      bool           `yaml:"keepAfter,omitempty"`
	SkipStoreState bool           `yaml:"skipStoreState,omitempty"`
}

func GetFrameworkConfig() (*TestConfig, error) {
	var fc TestConfig
	testconfig := os.Getenv("E2E_TESTCONFIG")
	if testconfig == "" {
		return nil, errors.New("Empty E2E_TESTCONFIG env var")
	}
	if _, err := os.Stat(testconfig); os.IsNotExist(err) {
		testconfigDir := os.Getenv("E2E_TESTCONFIG_DIR")
		if testconfigDir == "" {
			testconfigDir, _ = os.Getwd()
		}
		if _, err := os.Stat(fmt.Sprintf("%s/testconfigs/%s", testconfigDir, testconfig)); os.IsNotExist(err) {
			return nil, errors.Wrap(err, "failed to find test config file")
		}
		testconfig = fmt.Sprintf("%s/testconfigs/%s", testconfigDir, testconfig)
	}
	yamlFile, err := os.ReadFile(testconfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read test config file")
	}
	err = yaml.Unmarshal(yamlFile, &fc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal test config file")
	}

	e2eNs := os.Getenv("TEST_NAMESPACE")
	if e2eNs != "" {
		fc.Settings.Namespace = e2eNs
	}
	return &fc, nil
}

func GetConfigForTestCase(t *testing.T) map[string]string {
	for _, testCase := range TF.TestConfig.Settings.CaseSettings {
		if testCase.Name == t.Name() {
			return testCase.Config
		}
	}
	return nil
}
