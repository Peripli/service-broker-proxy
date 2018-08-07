package osb

import (
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/Peripli/service-broker-proxy/pkg/proxy"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/pkg/errors"

	"fmt"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// BusinessLogic provides an implementation of the pmorie/osb-broker-lib/pkg/broker/Interface interface.
type BusinessLogic struct {
	config *ClientConfig
}

// NewBusinessLogic creates an OSB business logic containing logic to proxy OSB calls
func NewBusinessLogic(config *ClientConfig) (*BusinessLogic, error) {
	return &BusinessLogic{
		config: config,
	}, nil
}

func (b *BusinessLogic) HandleRequest(rw http.ResponseWriter, req *http.Request) {
	target, err := osbClient(req, b.config)
	if err != nil {
		// TODO
		return
	}

	proxier := proxy.ReverseProxy()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		// TODO
		return
	}
	targetURL, _ := url.Parse(target.URL)

	reqBuilder := proxier.RequestBuilder().
		URL(targetURL).
		Auth(target.Username, target.Password)

	response, err := proxier.ProxyRequest(req, reqBuilder, body)
	if err != nil {
		// TODO
		return
	}

	rw.WriteHeader(response.StatusCode)
	rw.Write(response.Body)
}

func osbClient(request *http.Request, config *ClientConfig) (*Target, error) {
	vars := mux.Vars(request)
	brokerID, ok := vars["brokerID"]
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
