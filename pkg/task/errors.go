package task

import (
	"context"
	"errors"
	logging "github.com/ipfs/go-log/v2"
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

	if hasErrorMessage(err, "deal rejected: Price per byte too low") {
		return DealRejectedPriceTooLow
	}

	logger := logging.Logger("error-resolution")
	innerError := err
	for {
		logger.With("innerError", innerError).Debug("error resolution trace")
		innerError = errors.Unwrap(innerError)
		if innerError == nil {
			break
		}
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
