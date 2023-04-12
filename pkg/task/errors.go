package task

import (
	"context"
	"errors"
	"github.com/data-preservation-programs/RetrievalBot/pkg/requesterror"
	"strings"
)

type ErrorCode string

const (
	ErrorCodeNone                  ErrorCode = ""
	NoValidMultiAddrs              ErrorCode = "no_valid_multiaddrs"
	CannotConnect                  ErrorCode = "cannot_connect"
	NotFound                       ErrorCode = "not_found"
	RetrievalFailure               ErrorCode = "retrieval_failure"
	ProtocolNotSupported           ErrorCode = "protocol_not_supported"
	Timeout                        ErrorCode = "timeout"
	DealRejectedPricePerByteTooLow ErrorCode = "deal_rejected_price_per_byte_too_low"
	DealRejectedUnsealPriceTooLow  ErrorCode = "deal_rejected_unseal_price_too_low"
	Throttled                      ErrorCode = "throttled"
	NoAccess                       ErrorCode = "no_access"
	UnderMaintenance               ErrorCode = "under_maintenance"
	NotOnline                      ErrorCode = "not_online"
	UnconfirmedBlockTransfer       ErrorCode = "unconfirmed_block_transfer"
	CIDCodecNotSupported           ErrorCode = "cid_codec_not_supported"
	ResponseRejected               ErrorCode = "response_rejected"
	DealStateMissing               ErrorCode = "deal_state_missing"
)

var errorStringMap = map[string]ErrorCode{
	"deal rejected: Price per byte too low":                        DealRejectedPricePerByteTooLow,
	"deal rejected: Unseal price too small":                        DealRejectedUnsealPriceTooLow,
	"Too many retrieval deals received":                            Throttled,
	"Access Control":                                               NoAccess,
	"Under maintenance, retry later":                               UnderMaintenance,
	"deal rejected: miner is not accepting online retrieval deals": NotOnline,
	"unconfirmed block transfer":                                   UnconfirmedBlockTransfer,
	"no decoder registered for multicodec code":                    CIDCodecNotSupported,
	"not found in db":                                              NotFound,
	"response rejected":                                            ResponseRejected,
	"failed to fetch storage deal state":                           DealStateMissing,
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

	for s, code := range errorStringMap {
		if strings.Contains(err.Error(), s) {
			return code
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
