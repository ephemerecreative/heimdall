package authenticators

import (
	"context"

	"github.com/dadrus/heimdall/internal/heimdall"
	"github.com/dadrus/heimdall/internal/pipeline/interfaces"
)

type noopAuthenticator struct{}

func NewNoopAuthenticator() *noopAuthenticator {
	return &noopAuthenticator{}
}

func (*noopAuthenticator) Authenticate(_ context.Context, _ interfaces.AuthDataSource, sc *heimdall.SubjectContext) error {
	sc.Subject = &heimdall.Subject{}
	return nil
}