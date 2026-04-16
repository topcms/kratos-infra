package errors

import (
	stderrors "errors"
	"fmt"
)

var (
	ErrUnauthorized = stderrors.New("unauthorized")
	ErrForbidden    = stderrors.New("forbidden")
	ErrNotFound     = stderrors.New("not found")
	ErrInternal     = stderrors.New("internal error")
)

// BizError 用于封装可跨服务传递的业务错误。
type BizError struct {
	Code    int32
	Reason  string
	Message string
}

func (e *BizError) Error() string {
	return fmt.Sprintf("code=%d reason=%s message=%s", e.Code, e.Reason, e.Message)
}

func New(code int32, reason, message string) error {
	return &BizError{
		Code:    code,
		Reason:  reason,
		Message: message,
	}
}
