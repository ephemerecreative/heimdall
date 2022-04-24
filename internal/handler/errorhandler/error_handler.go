package errorhandler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/dadrus/heimdall/internal/heimdall"
	"github.com/dadrus/heimdall/internal/x"
)

func NewErrorHandler(verbose bool, logger zerolog.Logger) fiber.ErrorHandler {
	return x.IfThenElse(verbose, verboseErrorHandler(logger), defaultErrorHandler(logger))
}

func defaultErrorHandler(logger zerolog.Logger) fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		switch {
		case errors.Is(err, heimdall.ErrArgument):
			return ctx.SendStatus(fiber.StatusBadRequest)
		case errors.Is(err, heimdall.ErrAuthentication):
			return ctx.SendStatus(fiber.StatusUnauthorized)
		case errors.Is(err, heimdall.ErrAuthorization):
			return ctx.SendStatus(fiber.StatusForbidden)
		case errors.Is(err, heimdall.ErrCommunicationTimeout):
			return ctx.SendStatus(fiber.StatusBadGateway)
		case errors.Is(err, &heimdall.RedirectError{}):
			var redirectError *heimdall.RedirectError

			errors.As(err, &redirectError)

			return ctx.Redirect(redirectError.RedirectTo.String(), redirectError.Code)
		default:
			logger.Error().Err(err).Msg("Error occurred")

			return ctx.SendStatus(fiber.StatusInternalServerError)
		}
	}
}

func verboseErrorHandler(logger zerolog.Logger) fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		switch {
		case errors.Is(err, heimdall.ErrArgument):
			return ctx.Status(fiber.StatusBadRequest).Format(err)
		case errors.Is(err, heimdall.ErrAuthentication):
			return ctx.Status(fiber.StatusUnauthorized).Format(err)
		case errors.Is(err, heimdall.ErrAuthorization):
			return ctx.Status(fiber.StatusForbidden).Format(err)
		case errors.Is(err, heimdall.ErrCommunicationTimeout):
			return ctx.Status(fiber.StatusBadGateway).Format(err)
		case errors.Is(err, &heimdall.RedirectError{}):
			var redirectError *heimdall.RedirectError

			errors.As(err, &redirectError)

			return ctx.Redirect(redirectError.RedirectTo.String(), redirectError.Code)
		default:
			logger.Error().Err(err).Msg("Error occurred")

			return ctx.Status(fiber.StatusInternalServerError).Format(err)
		}
	}
}
