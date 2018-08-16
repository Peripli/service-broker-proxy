package osb

import (
	"net/http"

	"github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/sirupsen/logrus"
)

type BrFetcherImpl struct {
	Config *ClientConfig
	Tr     http.RoundTripper
}

var _ osb.BrokerFetcher = &BrFetcherImpl{}

func (b *BrFetcherImpl) Broker(request *web.Request, brokerID string) (*types.Broker, error) {
	broker := &types.Broker{
		BrokerURL: b.Config.URL + "/" + brokerID,
		Credentials: &types.Credentials{
			Basic: &types.Basic{
				Username: b.Config.Username,
				Password: b.Config.Password,
			},
		},
	}
	logrus.Debug("Building OSB client for broker with name: ", b.Config.Name, " accesible at: ", broker.BrokerURL)

	return broker, nil
}

func (b *BrFetcherImpl) RoundTrip(request *http.Request) (*http.Response, error) {
	return b.Tr.RoundTrip(request)
}
