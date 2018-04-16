package platform

// Client configuration provides the logic for configuring and creating a client to be used to perform platform specific operations
type ClientConfiguration interface {
	CreateFunc() (Client, error)
	Validate() error
}
