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
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	rookclient "github.com/rook/rook/pkg/client/clientset/versioned"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	cephlcmclient "github.com/Mirantis/pelagia/pkg/client/clientset/versioned/typed/ceph.pelagia.lcm/v1alpha1"
	lcmcommon "github.com/Mirantis/pelagia/pkg/common"
	lcmconfig "github.com/Mirantis/pelagia/pkg/controller/config"
)

var (
	TF         Framework
	StepNumber int
)

type Framework struct {
	ManagedCluster       *ManagedConfig
	InitialClusterState  *StoreState
	PreviousClusterState *StoreState
	TestConfig           *TestConfig
	Log                  zerolog.Logger
	E2eImage             string
}

type ManagedConfig struct {
	Context                 context.Context
	Client                  client.Client
	DynamicClient           dynamic.Interface
	KubeClient              *kubernetes.Clientset
	CephDplClient           cephlcmclient.CephDeploymentInterface
	CephDplSecretClient     cephlcmclient.CephDeploymentSecretInterface
	CephHealthClient        cephlcmclient.CephDeploymentHealthInterface
	CephOsdRemoveTaskClient cephlcmclient.CephOsdRemoveTaskInterface
	RookClientset           *rookclient.Clientset
	KubeConfig              *rest.Config
	LcmConfig               lcmconfig.LcmConfig
	LcmNamespace            string
	OpenstackClient         *OpenstackClient
}

func NewManagedCluster(cephdeployNamespace string, config *rest.Config) (*ManagedConfig, error) {
	managedCluster := &ManagedConfig{
		Context:       context.Background(),
		CephDplClient: nil,
	}
	var err error

	managedCluster.KubeConfig = config
	managedCluster.KubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot create kubernetes client from kubeconfig")
	}

	mapperClient, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot create mapper client from kubeconfig")
	}
	mapper, err := apiutil.NewDynamicRESTMapper(config, mapperClient)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot create mapper from kubeconfig")
	}
	crClient, err := client.New(config, client.Options{Mapper: mapper})
	if err != nil {
		return nil, errors.Wrap(err, "Cannot create controller-runtime client from kubeconfig")
	}

	managedCluster.Client = crClient
	managedCluster.DynamicClient, _ = dynamic.NewForConfig(config)
	managedCluster.RookClientset, err = rookclient.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot create rook clientset from kubeconfig")
	}

	cephDplClient, err := cephlcmclient.NewForConfig(managedCluster.KubeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot create cephdeployment client from kubeconfig")
	}
	cephLcmClient, err := cephlcmclient.NewForConfig(managedCluster.KubeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot create ceph-lcm client from kubeconfig")
	}

	_, cdErr := cephDplClient.CephDeployments(cephdeployNamespace).List(managedCluster.Context, metav1.ListOptions{})
	if cdErr != nil {
		if strings.Contains(cdErr.Error(), "the server could not find the requested resource") {
			return nil, errors.Wrap(cdErr, "no CephDeployment client found on the cluster")
		}
		return nil, errors.Wrap(cdErr, "Cannot get CephDeployment list from kubeconfig")
	}
	// CephDeployment CRD exists, create all related clients
	managedCluster.CephDplClient = cephDplClient.CephDeployments(cephdeployNamespace)
	managedCluster.CephDplSecretClient = cephDplClient.CephDeploymentSecrets(cephdeployNamespace)
	managedCluster.CephHealthClient = cephLcmClient.CephDeploymentHealths(cephdeployNamespace)
	managedCluster.CephOsdRemoveTaskClient = cephLcmClient.CephOsdRemoveTasks(cephdeployNamespace)

	cm, cmErr := managedCluster.GetConfigMap(lcmconfig.LcmConfigMapName, cephdeployNamespace)
	if cmErr != nil {
		return nil, errors.Wrap(cmErr, "Cannot get ConfigMap lcmconfig from kubeconfig")
	}

	lcmConfig := lcmconfig.ReadConfiguration(TF.Log, cm.Data)
	managedCluster.LcmNamespace = cephdeployNamespace
	managedCluster.LcmConfig = lcmConfig

	return managedCluster, nil
}

func Setup(fc *TestConfig) error {
	f, err := setupFramework(fc)
	if err != nil {
		return errors.Wrapf(err, "failed to set test environment")
	}
	TF = *f
	if fc.Settings.SkipStoreState {
		return nil
	}
	TF.InitialClusterState, err = NewStoreState()
	if err != nil {
		return errors.Wrap(err, "Cannot save current ceph cluster state as backup")
	}
	TF.InitialClusterState.CopyState(TF.PreviousClusterState)
	return nil
}

func setupFramework(fc *TestConfig) (*Framework, error) {
	f := &Framework{
		ManagedCluster: &ManagedConfig{
			Context:       context.Background(),
			CephDplClient: nil,
		},
		InitialClusterState:  &StoreState{},
		PreviousClusterState: &StoreState{},
	}
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	f.Log = lcmcommon.InitLogger(false)

	f.TestConfig = fc

	kubeconfig := os.Getenv("KUBECONFIG")
	if fc.Settings.KubeconfigFile != "" {
		kubeconfig = fc.Settings.KubeconfigFile
	} else if fc.Settings.KubeconfigURL != "" {
		err := GetKubeconfig(fc.Settings.KubeconfigURL, "../e2e_kubeconfig")
		if err != nil {
			return nil, errors.Wrap(err, "failed to get cluster kubeconfig from provided in test config url")
		}
		kubeconfig = "../e2e_kubeconfig"
	}
	if kubeconfig == "" {
		return nil, errors.New("Empty KUBECONFIG env var")
	}
	if !path.IsAbs(kubeconfig) {
		kubeconfig, _ = filepath.Abs(kubeconfig)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot build kube from kubeconfig")
	}

	managedCluster, err := NewManagedCluster(fc.Settings.Namespace, config)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot initialize managed cluster clients")
	}
	f.ManagedCluster = managedCluster

	f.E2eImage = os.Getenv("CEPH_E2E_IMAGE")
	return f, nil
}

func Teardown() error {
	state := TF.InitialClusterState
	err := state.RestoreStoredState()
	if err != nil {
		errMsg := fmt.Sprintf("Teardown failed: failed to restore initial cluster state: %v", err)
		TF.Log.Error().Err(err).Msg("")
		if state.ExportDir != "" {
			err := state.ExportStoreState(state.ExportDir)
			if err != nil {
				println(fmt.Sprintf("failed to export state to file: %v", err))
				println("printing state inline:")
				stateJSON, convErr := state.ConvertStateToJSON()
				if convErr == nil {
					println(fmt.Sprintf("%v", string(stateJSON)))
				}
			}
		} else {
			println("printing state inline:")
			stateJSON, convErr := state.ConvertStateToJSON()
			if convErr == nil {
				println(fmt.Sprintf("%v", string(stateJSON)))
			}
		}
		return errors.New(errMsg)
	}

	return nil
}

func BaseSetup(t *testing.T) error {
	t.Log("Setup started..")
	fc, err := GetFrameworkConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get test config")
	}
	if !lcmcommon.Contains(fc.Cases, t.Name()) {
		t.Logf("%s not in test cases list", t.Name())
		t.SkipNow()
	}
	err = Setup(fc)
	if err != nil {
		t.Logf("Setup failed: %v", err)
		return err
	}
	t.Log("Setup successfully done")
	return nil
}

func SetupTeardown(t *testing.T) func() {
	StepNumber = 0
	err := BaseSetup(t)
	if err != nil {
		t.Fatal(err)
	}
	return func() {
		StepNumber = 0
		if TF.TestConfig.Settings.KeepAfter {
			t.Log("Teardown skipped due to keepAfter flag enabled")
			return
		}

		t.Log("Teardown started..")
		err = Teardown()
		if err != nil {
			t.Logf("Teardown failed: %v", err)
			t.Fatal(err)
		}
		t.Log("Teardown successfully done")
	}
}

func SetupWithCustomTeardown(t *testing.T, customTeardown func() error) func() {
	StepNumber = 0
	err := BaseSetup(t)
	if err != nil {
		t.Fatal(err)
	}
	return func() {
		StepNumber = 0
		if TF.TestConfig.Settings.KeepAfter {
			t.Log("Teardown skipped due to keepAfter flag enabled")
			return
		}

		t.Log("Teardown started..")
		errs := make([]string, 0)
		err = customTeardown()
		if err != nil {
			errs = append(errs, fmt.Sprintf("Custom teardown function failed: %v", err))
		}
		err = Teardown()
		if err != nil {
			errs = append(errs, fmt.Sprintf("Teardown failed: %v", err))
		}
		if len(errs) > 0 {
			t.Fatalf("%v", strings.Join(errs, ", "))
		}
		t.Log("Teardown successfully done")
	}
}

func CustomTeardown(t *testing.T, customTeardown func() error) {
	StepNumber = 0
	if TF.TestConfig.Settings.KeepAfter {
		t.Log("Teardown skipped due to keepAfter flag enabled")
		return
	}

	t.Log("Teardown started..")
	errs := make([]string, 0)
	err := customTeardown()
	if err != nil {
		errs = append(errs, fmt.Sprintf("Custom teardown function failed: %v", err))
	}
	err = Teardown()
	if err != nil {
		errs = append(errs, fmt.Sprintf("Teardown failed: %v", err))
	}
	if len(errs) > 0 {
		t.Fatalf("%v", strings.Join(errs, ", "))
	}
	t.Log("Teardown successfully done")
}

func GetKubeconfig(url, fileName string) error {
	r, err := http.Get(url)
	if err != nil {
		return errors.Wrapf(err, "failed to get kubeconfig by URL: %v", url)
	}
	defer r.Body.Close()
	content, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.Wrap(err, "failed to parse kubeconfig URL body")
	}
	err = os.WriteFile(fileName, content, 0666)
	if err != nil {
		return errors.Wrapf(err, "failed to write kubeconfig to %v", fileName)
	}
	return nil
}

func Step(t *testing.T, msg string, args ...interface{}) {
	StepNumber++
	if len(args) > 0 {
		t.Logf("%v ## Step %d - %v", time.Now().UTC().String(), StepNumber, fmt.Sprintf(msg, args...))
	} else {
		t.Logf("%v ## Step %d - %v", time.Now().UTC().String(), StepNumber, msg)
	}
}
