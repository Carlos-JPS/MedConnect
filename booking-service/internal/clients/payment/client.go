package payment

type NoopClient struct{}

func NewNoopClient() *NoopClient {
	return &NoopClient{}
}
