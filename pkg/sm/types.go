package sm

import "encoding/json"
import osbc "github.com/pmorie/go-open-service-broker-client/v2"

// BrokerList type used for responses from the Service Manager client
type BrokerList struct {
	Brokers []Broker `json:"brokers"`
}

// Broker type used for responses from the Service Manager client
type Broker struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	BrokerURL   string                     `json:"broker_url"`
	Credentials *Credentials               `json:"credentials,omitempty"`
	Catalog     *osbc.CatalogResponse      `json:"catalog"`
	Metadata    map[string]json.RawMessage `json:"metadata,omitempty"`
}

// Basic type for representing basic authorization credentials
type Basic struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Credentials type for representing broker credentials
type Credentials struct {
	Basic *Basic `json:"basic,omitempty"`
}
