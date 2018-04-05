package platform

type Client interface {
	GetBrokers() (*ServiceBrokerList, error)
	CreateBroker(r *CreateServiceBrokerRequest) (*ServiceBroker, error)
	DeleteBroker(r *DeleteServiceBrokerRequest) error
	UpdateBroker(r *UpdateServiceBrokerRequest) (*ServiceBroker, error)
}
