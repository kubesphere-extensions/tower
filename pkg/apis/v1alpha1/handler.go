package v1alpha1

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/emicklei/go-restful/v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/klog/v2"
	clusterv1alpha1 "kubesphere.io/api/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type handler struct {
	client client.Reader

	yamlPrinter *printers.YAMLPrinter
}

func newHandler(client client.Reader) *handler {
	return &handler{
		client:      client,
		yamlPrinter: &printers.YAMLPrinter{},
	}
}

// generateAgentDeployment will return a deployment yaml for proxy connection type cluster
// ProxyPublishAddress takes high precedence over proxyPublishService, use proxyPublishService ingress
// address only when proxyPublishAddress is not provided.
func (h *handler) generateAgentDeployment(request *restful.Request, response *restful.Response) {
	clusterName := request.QueryParameter("cluster")

	cluster := &clusterv1alpha1.Cluster{}
	if err := h.client.Get(context.Background(), client.ObjectKey{Name: clusterName}, cluster); err != nil {
		klog.Error(err)
		response.WriteError(http.StatusInternalServerError, err) // nolint
		return
	}

	if cluster.Spec.Connection.Type != clusterv1alpha1.ConnectionTypeProxy {
		err := fmt.Sprintf("cluster %s is not using proxy connection", cluster.Name)
		klog.Error(err)
		response.WriteErrorString(http.StatusBadRequest, err) // nolint
		return
	}

	proxyAddress, agentImage, err := h.populateProxyAddressAndImage()
	if err != nil {
		klog.Error(err)
		response.WriteError(http.StatusInternalServerError, err) // nolint
		return
	}

	var buf bytes.Buffer
	if err = h.generateDefaultDeployment(cluster, proxyAddress, agentImage, &buf); err != nil {
		klog.Error(err)
		response.WriteError(http.StatusInternalServerError, err) // nolint
		return
	}

	response.Write(buf.Bytes()) // nolint
}

func (h *handler) populateProxyAddressAndImage() (string, string, error) {
	kubeSphereConfigMap := &corev1.ConfigMap{}
	if err := h.client.Get(context.Background(), client.ObjectKey{Namespace: kubeSphereNamespace, Name: kubeSphereConfigName}, kubeSphereConfigMap); err != nil {
		return "", "", err
	}
	kubeSphereConfig, err := getFromConfigMap(kubeSphereConfigMap)
	if err != nil {
		return "", "", err
	}

	if kubeSphereConfig.MultiClusterOptions.ProxyPublishAddress != "" {
		return kubeSphereConfig.MultiClusterOptions.ProxyPublishAddress, kubeSphereConfig.MultiClusterOptions.AgentImage, nil
	}

	if kubeSphereConfig.MultiClusterOptions.ProxyPublishService == "" {
		return "", "", fmt.Errorf("neither proxy address nor proxy service provided")
	}

	// use service ingress address
	namespace := kubeSphereNamespace
	parts := strings.Split(kubeSphereConfig.MultiClusterOptions.ProxyPublishService, ".")
	if len(parts) > 1 && len(parts[1]) != 0 {
		namespace = parts[1]
	}

	service := &corev1.Service{}
	if err = h.client.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: parts[0]}, service); err != nil {
		return "", "", fmt.Errorf("service %s not found in namespace %s", parts[0], namespace)
	}

	if len(service.Spec.Ports) == 0 {
		return "", "", fmt.Errorf("there are no ports in proxy service %s spec", kubeSphereConfig.MultiClusterOptions.ProxyPublishService)
	}

	port := service.Spec.Ports[0].Port

	var serviceAddress string
	for _, ingress := range service.Status.LoadBalancer.Ingress {
		if len(ingress.Hostname) != 0 {
			serviceAddress = fmt.Sprintf("http://%s:%d", ingress.Hostname, port)
		}

		if len(ingress.IP) != 0 {
			serviceAddress = fmt.Sprintf("http://%s:%d", ingress.IP, port)
		}
	}

	if len(serviceAddress) == 0 {
		return "", "", fmt.Errorf("cannot generate agent deployment yaml for member cluster "+
			" because %s service has no public address, please check %s status, or set address "+
			" mannually in ClusterConfiguration", kubeSphereConfig.MultiClusterOptions.ProxyPublishService, kubeSphereConfig.MultiClusterOptions.ProxyPublishService)
	}

	return serviceAddress, kubeSphereConfig.MultiClusterOptions.AgentImage, nil
}

func (h *handler) generateDefaultDeployment(cluster *clusterv1alpha1.Cluster, proxyAddress, agentImage string, w io.Writer) error {
	if _, err := url.Parse(proxyAddress); err != nil {
		return fmt.Errorf("invalid proxy address %s, should format like http[s]://1.2.3.4:123", proxyAddress)
	}

	if cluster.Spec.Connection.Type == clusterv1alpha1.ConnectionTypeDirect {
		return fmt.Errorf("cluster is not using proxy connection")
	}

	namespace := corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: kubeSphereNamespace,
		},
	}
	serviceAccount := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tower",
			Namespace: kubeSphereNamespace,
		},
	}
	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "tower",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      "tower",
				Namespace: kubeSphereNamespace,
			},
		},
	}
	agent := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-agent",
			Namespace: kubeSphereNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":                       "agent",
					"app.kubernetes.io/part-of": "tower",
				},
			},
			Strategy: appsv1.DeploymentStrategy{},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                       "agent",
						"app.kubernetes.io/part-of": "tower",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "agent",
							Command: []string{
								"/agent",
								fmt.Sprintf("--name=%s", cluster.Name),
								fmt.Sprintf("--token=%s", cluster.Spec.Connection.Token),
								fmt.Sprintf("--proxy-server=%s", proxyAddress),
								"--keepalive=10s",
								"--kubesphere-service=ks-apiserver.kubesphere-system.svc:80",
								"--kubernetes-service=kubernetes.default.svc:443",
								"--v=0",
							},
							Image: agentImage,
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("200M"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100M"),
								},
							},
						},
					},
					ServiceAccountName: "tower",
				},
			},
		},
	}

	if err := h.yamlPrinter.PrintObj(&namespace, w); err != nil {
		return err
	}
	if err := h.yamlPrinter.PrintObj(&serviceAccount, w); err != nil {
		return err
	}
	if err := h.yamlPrinter.PrintObj(&clusterRoleBinding, w); err != nil {
		return err
	}
	return h.yamlPrinter.PrintObj(&agent, w)
}
