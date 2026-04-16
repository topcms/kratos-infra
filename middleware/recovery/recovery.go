package recovery

import (
	"context"
	"fmt"

	infraerrors "kratos-infra/errors"

	"github.com/go-kratos/kratos/v2/middleware"
)

func Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (_ any, err error) {
			defer func() {
				if p := recover(); p != nil {
					err = fmt.Errorf("%w: panic=%v", infraerrors.ErrInternal, p)
				}
			}()

			return handler(ctx, req)
		}
	}
}
