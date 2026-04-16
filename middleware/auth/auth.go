package auth

import (
	"context"
	"strings"

	infraerrors "kratos-infra/errors"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

const authorizationHeader = "Authorization"

type TokenValidator func(ctx context.Context, token string) error

func Server(validate TokenValidator) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if validate == nil {
				return nil, infraerrors.ErrInternal
			}

			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, infraerrors.ErrUnauthorized
			}

			token := parseBearerToken(tr.RequestHeader().Get(authorizationHeader))
			if token == "" {
				return nil, infraerrors.ErrUnauthorized
			}

			if err := validate(ctx, token); err != nil {
				return nil, infraerrors.ErrUnauthorized
			}

			return handler(ctx, req)
		}
	}
}

func parseBearerToken(value string) string {
	if value == "" {
		return ""
	}

	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}
