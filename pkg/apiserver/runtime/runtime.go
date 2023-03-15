package runtime

import (
	"fmt"

	"github.com/emicklei/go-restful/v3"
)

func NewWebService(version string) *restful.WebService {
	webservice := new(restful.WebService)
	webservice.Path(fmt.Sprintf("/apis/%s", version)).Produces(restful.MIME_JSON)
	return webservice
}
