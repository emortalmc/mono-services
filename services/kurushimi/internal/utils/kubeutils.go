package utils

import (
	"errors"
	"flag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

func CreateKubernetesConfig() (*rest.Config, error) {
	var config *rest.Config
	var err error

	config, err = rest.InClusterConfig()
	if !errors.Is(err, rest.ErrNotInCluster) {
		return config, err
	}

	var kubeConfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeConfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeConfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err = clientcmd.BuildConfigFromFlags("", *kubeConfig)

	return config, err
}
