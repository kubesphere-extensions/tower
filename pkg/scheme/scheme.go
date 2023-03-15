package scheme

import (
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1alpha1 "kubesphere.io/api/cluster/v1alpha1"
)

// Scheme contains all types of custom Scheme and kubernetes client-go Scheme.
var Scheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(Scheme)
	_ = clusterv1alpha1.AddToScheme(Scheme)
}
