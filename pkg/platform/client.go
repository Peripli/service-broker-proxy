package platform

type Client interface {
	GetBrokers() (*ServiceBrokerList, error)
	CreateBrokers(r *CreateServiceBrokerRequest) (*ServiceBroker, error)
	DeleteBrokers(r *DeleteServiceBrokerRequest) error
	UpdateBrokers(r *UpdateServiceBrokerRequest) (*ServiceBroker, error)
}
