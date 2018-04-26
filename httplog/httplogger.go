package httplogger

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Transport implements http.RoundTripper. When set as Transport of http.Client, it executes HTTP requests with logging.
// No field is mandatory.
type Transport struct {
	Transport http.RoundTripper
	LogFunc   func(resp *http.Response, req *http.Request)
}

// THe default logging transport that wraps http.DefaultTransport.
var DefaultTransport = &Transport{
	Transport: http.DefaultTransport,
}

// Used if transport.LogRequest is not set.
var DefaultLogFunc = func(resp *http.Response, req *http.Request) {
	fmt.Printf("Request : %s %s \n", req.Method, req.URL)
	fmt.Printf("Response : %s \n", resp.Status)
}

type contextKey struct {
	name string
}

var ContextKeyRequestStart = &contextKey{"RequestStart"}

// ReadBody from request and restore it for previous state
func readBody(req *http.Request) []byte {
	var bodyBuffer []byte
	if req.Body != nil {
		bodyBuffer, _ = ioutil.ReadAll(req.Body) // after this operation body will equal 0
		// Restore the io.ReadCloser to request
		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBuffer))
	}

	return bodyBuffer
}

// RoundTrip is the core part of this module and implements http.RoundTripper.
// Executes HTTP request with request/response logging.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := context.WithValue(req.Context(), ContextKeyRequestStart, time.Now())
	req = req.WithContext(ctx)

	// read body
	bodyBuffer := readBody(req)

	resp, err := t.transport().RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Restore the io.ReadCloser to logger
	req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBuffer))
	t.logFunc(resp, req)

	return resp, err
}

func (t *Transport) logFunc(resp *http.Response, req *http.Request) {
	if t.LogFunc != nil {
		t.LogFunc(resp, req)
	}

	DefaultLogFunc(resp, req)
}

func (t *Transport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}

	return http.DefaultTransport
}
