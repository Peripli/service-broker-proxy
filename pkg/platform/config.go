package platform

type ClientConfiguration interface {
	CreateFunc() (Client, error)
	Validate() error
}
