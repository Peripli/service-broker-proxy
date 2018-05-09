package sm

// BrokerList type used for responses from the Service Manager client
type BrokerList struct {
	Brokers []Broker `json:"brokers"`
}

// Broker type used for responses from the Service Manager client
type Broker struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	BrokerURL   string       `json:"broker_url"`
	Credentials *Credentials `json:"credentials,omitempty"`
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
