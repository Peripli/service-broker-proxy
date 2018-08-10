package osb

import (
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	smOsb "github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/proxy"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/pkg/errors"

	"fmt"

	"github.com/sirupsen/logrus"
)

type osbController struct {
	internalController *smOsb.Controller
	proxier            *proxy.Proxy

	config *ClientConfig
}

// NewOsbController creates an OSB business logic containing logic to proxy OSB calls
func NewOsbController(config *ClientConfig) (*osbController, error) {

	controller := &osbController{
		config: config,
	}
	controller.internalController = &smOsb.Controller{Handler: controller.handler}

	defaultTransport := http.DefaultTransport.(*http.Transport)
	t := &http.Transport{
		Proxy:                 defaultTransport.Proxy,
		TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
		ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
	}
	t.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: config.Insecure,
	}
	t.DialContext = (&net.Dialer{
		Timeout:   time.Duration(config.TimeoutSeconds),
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext

	t.TLSClientConfig = &tls.Config{InsecureSkipVerify: config.Insecure}
	proxier := proxy.NewReverseProxy(proxy.Options{
		Transport: t,
	})
	controller.proxier = proxier

	return controller, nil
}

func (c *osbController) Routes() []web.Route {
	return c.internalController.Routes()
}

func (c *osbController) handler(request *web.Request) (*web.Response, error) {
	target, err := osbClient(request, c.config)
	if err != nil {
		return nil, err
	}

	targetURL, _ := url.Parse(target.URL)
	targetURL.Path = request.Request.URL.Path

	reqBuilder := c.proxier.RequestBuilder().
		URL(targetURL).
		Auth(target.Username, target.Password)

	resp, err := c.proxier.ProxyRequest(request.Request, reqBuilder, request.Body)
	if err != nil {
		return nil, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	webResponse := &web.Response{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       respBody,
	}

	return webResponse, nil
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
