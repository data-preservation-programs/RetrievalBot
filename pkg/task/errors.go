package task

import (
	"context"
	"errors"
	logging "github.com/ipfs/go-log/v2"
)

type ErrorCode string

const (
	ErrorCodeNone           ErrorCode = ""
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

	if hasErrorMessage(err, "deal rejected: Price per byte too low") {
		return DealRejectedPriceTooLow
	}

	logger := logging.Logger("error-resolution")
	logger.With("err", err).Debug("error resolution debug trace")
	for {
		innerError := errors.Unwrap(err)
		if innerError == nil {
			break
		}
		logger.With("innerError", innerError).Debug("error resolution debug trace")
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
