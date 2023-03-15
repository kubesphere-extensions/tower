package v1alpha1

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubesphere-extensions/tower/pkg/apiserver/runtime"
)

const version = "v1alpha1"

func AddToContainer(c *restful.Container, cache client.Reader) {
	webservice := runtime.NewWebService(version)

	h := newHandler(cache)
	webservice.Route(webservice.GET("/deployment").
		Doc("Return deployment yaml for cluster agent.").
		Param(webservice.QueryParameter("cluster", "Name of the cluster.").Required(true)).
		To(h.generateAgentDeployment).
		Returns(http.StatusOK, "ok", nil))

	c.Add(webservice)
}
