package testenv

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/testing_frameworks/integration"
)

const (
	envKubebuilderPath      = "K8S_ASSETS"
	defaultStartStopTimeout = 20 * time.Second
)

func defaultAssetPath(binary string) string {
	assetPath := os.Getenv(envKubebuilderPath)
	if assetPath == "" {
		panic("K8S_ASSETS env isn't set")
	}
	return filepath.Join(assetPath, binary)
}

var DefaultKubeAPIServerFlags = []string{
	"--etcd-servers={{ if .EtcdURL }}{{ .EtcdURL.String }}{{ end }}",
	"--cert-dir={{ .CertDir }}",
	"--insecure-port={{ if .URL }}{{ .URL.Port }}{{ end }}",
	"--insecure-bind-address={{ if .URL }}{{ .URL.Hostname }}{{ end }}",
	"--secure-port=0",
	"--admission-control=AlwaysAdmit",
}

type Environment struct {
	ControlPlane       integration.ControlPlane
	Config             *rest.Config
	KubeAPIServerFlags []string
}

func NewEnv(apiServerFlags []string) *Environment {
	return &Environment{
		KubeAPIServerFlags: apiServerFlags,
	}
}

func (e *Environment) Stop() error {
	return e.ControlPlane.Stop()
}

func (e Environment) getAPIServerFlags() []string {
	if len(e.KubeAPIServerFlags) == 0 {
		return DefaultKubeAPIServerFlags
	}
	return e.KubeAPIServerFlags
}

func (e *Environment) Start() error {
	e.ControlPlane = integration.ControlPlane{}
	e.ControlPlane.APIServer = &integration.APIServer{Args: e.getAPIServerFlags()}
	e.ControlPlane.Etcd = &integration.Etcd{}

	e.ControlPlane.APIServer.Path = defaultAssetPath("kube-apiserver")
	e.ControlPlane.Etcd.Path = defaultAssetPath("etcd")
	if err := os.Setenv("TEST_ASSET_KUBECTL", defaultAssetPath("kubectl")); err != nil {
		return err
	}

	e.ControlPlane.Etcd.StartTimeout = defaultStartStopTimeout
	e.ControlPlane.Etcd.StopTimeout = defaultStartStopTimeout
	e.ControlPlane.APIServer.StartTimeout = defaultStartStopTimeout
	e.ControlPlane.APIServer.StopTimeout = defaultStartStopTimeout

	if err := e.startControlPlane(); err != nil {
		return err
	}

	e.Config = &rest.Config{
		Host: e.ControlPlane.APIURL().Host,
	}
	return nil
}

func (e *Environment) startControlPlane() error {
	numTries, maxRetries := 0, 5
	for ; numTries < maxRetries; numTries++ {
		err := e.ControlPlane.Start()
		if err == nil {
			break
		}
	}

	if numTries == maxRetries {
		return fmt.Errorf("failed to start the controlplane. retried %d times", numTries)
	}
	return nil
}
