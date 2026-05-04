package availability

type NoopClient struct{}

func NewNoopClient() *NoopClient {
	return &NoopClient{}
}
