package infrafx

import (
	"github.com/dadrus/heimdall/internal/pipeline"
	"go.uber.org/fx"

	"github.com/dadrus/heimdall/internal/config"
	"github.com/dadrus/heimdall/internal/logging"
)

var Module = fx.Options(
	config.Module,
	logging.Module,
	pipeline.Module,
)