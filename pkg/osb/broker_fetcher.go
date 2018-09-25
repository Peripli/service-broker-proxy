package osb

import (
	"context"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/api/osb"
)


// BrokerDetails implements osb.BrokerRoundTripper
type BrokerDetails struct {
	Username string
	Password string
	URL      string

}

var _ osb.BrokerFetcher = &BrokerDetails{}

// FetchBroker implements osb.BrokerRoundTripper and returns the coordinates of the broker with the specified id
func (b *BrokerDetails) FetchBroker(ctx context.Context, brokerID string) (*types.Broker, error) {
	return &types.Broker{
		BrokerURL: b.URL + "/" + brokerID,
		Credentials: &types.Credentials{
			Basic: &types.Basic{
				Username: b.Username,
				Password: b.Password,
			},
		},
	}, nil
}
