package osb

import (
	"net/http"
	"net/url"

	"github.com/Peripli/service-broker-proxy/pkg/proxy"
	smOsb "github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/pkg/errors"

	"fmt"

	"github.com/sirupsen/logrus"
)

type myOsbController struct {
	internalController *smOsb.Controller

	config *ClientConfig
}

// NewOsbController creates an OSB business logic containing logic to proxy OSB calls
func NewOsbController(config *ClientConfig) (*myOsbController, error) {

	myOsb := &myOsbController{
		config: config,
	}
	myOsb.internalController = &smOsb.Controller{Handler: myOsb.handler}

	return myOsb, nil
}

func (c *myOsbController) Routes() []web.Route {
	return c.internalController.Routes()
}

func (c *myOsbController) handler(request *web.Request) (*web.Response, error) {
	logrus.Info(">>>>>>>>>>>>>>>>>>>>here")
	target, err := osbClient(request, c.config)
	if err != nil {
		return nil, err
	}

	proxier := proxy.ReverseProxy()

	targetURL, _ := url.Parse(target.URL)
	targetURL.Path = request.Request.URL.Path

	reqBuilder := proxier.RequestBuilder().
		URL(targetURL).
		Auth(target.Username, target.Password)

	response, err := proxier.ProxyRequest(request.Request, reqBuilder, request.Body)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func osbClient(request *web.Request, config *ClientConfig) (*Target, error) {
	brokerID, ok := request.PathParams["brokerID"]
	if !ok {
		errMsg := fmt.Sprintf("brokerId path parameter missing from %s", request.Host)
		logrus.WithError(errors.New(errMsg)).Error("Error building OSB client for proxy business logic")

		return nil, &util.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Description: errMsg,
		}
	}

	target := &Target{
		URL:      config.URL + "/" + brokerID,
		Username: config.Username,
		Password: config.Password,
	}

	logrus.Debug("Building OSB client for broker with name: ", config.Name, " accesible at: ", target.URL)
	return target, nil
}
