package net

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

type HTTPClient struct {
	timeout time.Duration
}

func NewHTTPClient(timeout time.Duration) HTTPClient {
	return HTTPClient{
		timeout: timeout,
	}
}

func (c HTTPClient) RetrievePiece(
	parent context.Context,
	host string,
	cid cid.Cid,
	length int64) (*task.RetrievalResult, error) {
	urlStr := host
	if urlStr[len(urlStr)-1] != '/' {
		urlStr += "/"
	}

	urlStr += "piece/" + cid.String()
	fileURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse url")
	}

	client := &http.Client{
		Timeout: c.timeout,
	}

	ctx, cancel := context.WithTimeout(parent, c.timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL.String(), nil)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	startTime := time.Now()
	resp, err := client.Do(request)
	if err != nil {
		return task.NewErrorRetrievalResult(task.CannotConnect, err), nil
	}

	fbTime := time.Since(startTime)

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return task.NewErrorRetrievalResult(task.NotFound, errors.Errorf("status code: %d", resp.StatusCode)), nil
	}

	if resp.StatusCode > 299 {
		return task.NewErrorRetrievalResult(task.RetrievalFailure, errors.Errorf("status code: %d", resp.StatusCode)), nil
	}

	downloaded, err := io.CopyN(io.Discard, resp.Body, length)
	if err != nil {
		return task.NewErrorRetrievalResult(task.RetrievalFailure, err), nil
	}

	elapsed := time.Since(startTime)
	return task.NewSuccessfulRetrievalResult(fbTime, downloaded, elapsed), nil
}
