package platform

// Client provides the logic for calling into the underlying platform and performing platform specific operations
type Client interface {
	GetBrokers() (*ServiceBrokerList, error)
	CreateBroker(r *CreateServiceBrokerRequest) (*ServiceBroker, error)
	DeleteBroker(r *DeleteServiceBrokerRequest) error
	UpdateBroker(r *UpdateServiceBrokerRequest) (*ServiceBroker, error)
}
