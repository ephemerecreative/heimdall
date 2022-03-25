package extractors

import (
	"errors"

	"github.com/dadrus/heimdall/internal/pipeline/interfaces"
)

type CompositeExtractStrategy []AuthDataExtractStrategy

func (ce CompositeExtractStrategy) GetAuthData(s interfaces.AuthDataSource) (string, error) {
	for _, e := range ce {
		if t, err := e.GetAuthData(s); err == nil {
			return t, nil
		}
	}

	return "", errors.New("no authentication data present")
}