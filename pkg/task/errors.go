package task

import (
	"context"
	"errors"
	"github.com/data-preservation-programs/RetrievalBot/pkg/requesterror"
)

type ErrorCode string

const (
	ErrorCodeNone           ErrorCode = ""
	NoValidMultiAddrs       ErrorCode = "no_valid_multiaddrs"
	CannotConnect           ErrorCode = "cannot_connect"
	NotFound                ErrorCode = "not_found"
	RetrievalFailure        ErrorCode = "retrieval_failure"
	ProtocolNotSupported    ErrorCode = "protocol_not_supported"
	Timeout                 ErrorCode = "timeout"
	DealRejectedPriceTooLow ErrorCode = "deal_rejected_price_too_low"
)

func hasErrorMessage(err error, msg string) bool {
	if err.Error() == msg {
		return true
	}

	inner := errors.Unwrap(err)
	if inner == nil {
		return false
	}

	return hasErrorMessage(inner, msg)
}

func resolveError(err error) ErrorCode {
	if errors.Is(err, context.DeadlineExceeded) {
		return Timeout
	}

	if errors.As(err, &requesterror.CannotConnectError{}) {
		return CannotConnect
	}

	if errors.As(err, &requesterror.InvalidIPError{}) ||
		errors.As(err, &requesterror.BogonIPError{}) ||
		errors.As(err, &requesterror.NoValidMultiAddrError{}) ||
		errors.As(err, &requesterror.HostLookupError{}) {
		return NoValidMultiAddrs
	}

	if errors.As(err, &requesterror.StreamError{}) {
		return RetrievalFailure
	}

	if hasErrorMessage(err, "deal rejected: Price per byte too low") {
		return DealRejectedPriceTooLow
	}

	return ErrorCodeNone
}

func resolveErrorResult(err error) *RetrievalResult {
	code := resolveError(err)
	if code == ErrorCodeNone {
		return nil
	}

	return NewErrorRetrievalResult(code, err)
}
