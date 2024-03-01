package v1alpha1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

const (
	kubeSphereNamespace        = "kubesphere-system"
	kubeSphereConfigName       = "kubesphere-config"
	kubeSphereConfigMapDataKey = "kubesphere.yaml"
)

type Config struct {
	MultiClusterOptions *MultiClusterOptions `json:"multicluster,omitempty" yaml:"multicluster,omitempty"`
}

type MultiClusterOptions struct {
	ProxyPublishService string `json:"proxyPublishService,omitempty" yaml:"proxyPublishService,omitempty"`
	ProxyPublishAddress string `json:"proxyPublishAddress,omitempty" yaml:"proxyPublishAddress,omitempty"`
	AgentImage          string `json:"agentImage,omitempty" yaml:"agentImage,omitempty"`
}

func getFromConfigMap(cm *corev1.ConfigMap) (*Config, error) {
	c := &Config{}
	value, ok := cm.Data[kubeSphereConfigMapDataKey]
	if !ok {
		return nil, fmt.Errorf("failed to get configmap kubesphere.yaml value")
	}

	if err := yaml.Unmarshal([]byte(value), c); err != nil {
		return nil, err
	}
	if c.MultiClusterOptions.AgentImage == "" {
		// The default value that is consistent with the image version in the chart,
		// which can reduce user configuration items in most scenarios.
		c.MultiClusterOptions.AgentImage = "kubesphere/tower:v0.2.1"
	}
	return c, nil
}
