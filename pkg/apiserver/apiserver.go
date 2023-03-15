package apiserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	clientcmd "github.com/iawia002/lia/kubernetes/client"
	genericclient "github.com/iawia002/lia/kubernetes/client/generic"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/kubesphere-extensions/tower/pkg/apis/v1alpha1"
	"github.com/kubesphere-extensions/tower/pkg/scheme"
)

func init() {
	restful.RegisterEntityAccessor("application/merge-patch+json", restful.NewEntityAccessorJSON(restful.MIME_JSON))
	restful.RegisterEntityAccessor("application/json-patch+json", restful.NewEntityAccessorJSON(restful.MIME_JSON))
}

type APIServer struct {
	server    *http.Server
	container *restful.Container

	client client.Client
}

func New(port uint, kubeconfig string) (*APIServer, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	client, err := genericclient.NewClient(config, genericclient.WithScheme(scheme.Scheme))
	if err != nil {
		return nil, err
	}

	s := &APIServer{
		server: &http.Server{
			Addr: fmt.Sprintf(":%d", port),
		},
		container: restful.NewContainer(),
		client:    client,
	}

	return s, nil
}

func (s *APIServer) installAPIs() {
	s.container.Filter(logging)

	// add health check APIs
	s.container.Handle("/healthz", healthz.CheckHandler{Checker: healthz.Ping})
	s.container.Handle("/readyz", healthz.CheckHandler{Checker: healthz.Ping})

	v1alpha1.AddToContainer(s.container, s.client)
}

func (s *APIServer) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		_ = s.server.Shutdown(ctx)
	}()

	s.installAPIs()
	s.server.Handler = s.container

	klog.Infof("Start listening on %s", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
