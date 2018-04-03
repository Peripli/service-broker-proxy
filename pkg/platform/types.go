package platform

type CreateServiceBrokerRequest struct {
	Name      string `json:"name"`
	BrokerURL string `json:"broker_url"`
	Username  string `json:"auth_username"`
	Password  string `json:"auth_password"`
	SpaceGUID string `json:"space_guid,omitempty"`
}

type UpdateServiceBrokerRequest struct {
	Guid      string `json:"guid"`
	Name      string `json:"name"`
	BrokerURL string `json:"broker_url"`
	Username  string `json:"auth_username"`
	Password  string `json:"auth_password"`
}

type DeleteServiceBrokerRequest struct {
	Guid string `json:"guid"`
}

type ServiceBroker struct {
	Guid      string `json:"guid"`
	Name      string `json:"name"`
	BrokerURL string `json:"broker_url"`
	Username  string `json:"auth_username"`
	Password  string `json:"auth_password"`
}

type ServiceBrokerList struct {
	ServiceBrokers []ServiceBroker `json:"service_brokers"`
}
