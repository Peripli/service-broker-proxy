package sm

import (
	"encoding/json"

	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

// Brokers type used for responses from the Service Manager client
type Brokers struct {
	Brokers []Broker `json:"brokers"`
}

// Broker type used for responses from the Service Manager client
type Broker struct {
	ID          string          `json:"id"`
	BrokerURL   string          `json:"broker_url"`

	Catalog  *osbc.CatalogResponse      `json:"catalog"`
	Metadata map[string]json.RawMessage `json:"metadata,omitempty"`
}
