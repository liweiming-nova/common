package register

type Register interface {
	Register(serviceName string, address string, metadata map[string]string) error
	Unregister(serviceName string, address string) error
}
