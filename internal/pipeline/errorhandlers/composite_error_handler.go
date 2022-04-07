package errorhandlers

import (
	"github.com/dadrus/heimdall/internal/heimdall"
	"github.com/dadrus/heimdall/internal/x/errorchain"
)

type CompositeErrorHandler []ErrorHandler

func (ceh CompositeErrorHandler) HandleError(ctx heimdall.Context, e error) (err error) {
	for _, eh := range ceh {
		err = eh.HandleError(ctx, e)
		if err != nil {
			// try next
			continue
		} else {
			return nil
		}
	}

	return err
}

func (CompositeErrorHandler) WithConfig(_ map[string]any) (ErrorHandler, error) {
	return nil, errorchain.NewWithMessage(heimdall.ErrConfiguration, "reconfiguration not allowed")
}