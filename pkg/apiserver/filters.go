package apiserver

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/klog/v2"
)

func logging(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	chain.ProcessFilter(req, resp)
	statusCode := resp.StatusCode()
	msg := fmt.Sprintf("%s %s %d", req.Request.Method, req.Request.URL, statusCode)
	switch statusCode {
	case http.StatusInternalServerError:
		klog.Error(msg)
	default:
		klog.Info(msg)
	}
}
