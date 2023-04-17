package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/PuerkitoBio/rehttp"

	"github.com/txsvc/apikit/config"
	"github.com/txsvc/apikit/logger"
	"github.com/txsvc/apikit/settings"

	"github.com/txsvc/stdlib/v2"
)

const (
	// format error messages
	MsgStatus = "%s. status: %d"
)

var (
	// ErrMissingCredentials indicates that a credentials are is missing
	ErrMissingCredentials = errors.New("missing credentials")

	// ErrApiInvocationError indicates an error in an API call
	ErrApiInvocationError = errors.New("api invocation error")

	contextKeyRequestStart = &contextKey{"RequestStart"}
)

// RestClient - API client encapsulating the http client
type (
	// StatusObject is used to report operation status and errors in an API request.
	// The struct can be used as a response object or be treated as an error object
	StatusObject struct {
		Status       int    `json:"status"`
		Message      string `json:"message"`
		ErrorMessage string `json:"error"`
		RootError    error  `json:"-"`
	}

	RestClient struct {
		HttpClient *http.Client
		Settings   *settings.DialSettings
		Logger     logger.Logger
		Trace      string
	}

	loggingTransport struct {
		InnerTransport http.RoundTripper
		Logger         logger.Logger
	}

	contextKey struct {
		name string
	}
)

func NewRestClient(ds *settings.DialSettings, logger logger.Logger) (*RestClient, error) {
	var _ds *settings.DialSettings

	httpClient := NewTransport(logger, http.DefaultTransport)

	// create or clone the settings
	if ds != nil {
		c := ds.Clone()
		_ds = &c
	} else {
		_ds = config.GetConfig().Settings()
		if _ds.Credentials == nil {
			_ds.Credentials = &settings.Credentials{} // just provide something to prevent NPEs further down
		}
	}

	return &RestClient{
		HttpClient: httpClient,
		Settings:   _ds,
		Logger:     logger,
		Trace:      stdlib.GetString(config.ForceTraceENV, ""),
	}, nil // FIXME: nothing creates an error here, remove later?
}

// GET is used to request data from the API. No payload, only queries!
func (c *RestClient) GET(uri string, response interface{}) (int, error) {
	return c.request("GET", fmt.Sprintf("%s%s", c.Settings.Endpoint, uri), nil, response)
}

func (c *RestClient) POST(uri string, request, response interface{}) (int, error) {
	return c.request("POST", fmt.Sprintf("%s%s", c.Settings.Endpoint, uri), request, response)
}

func (c *RestClient) PUT(uri string, request, response interface{}) (int, error) {
	return c.request("PUT", fmt.Sprintf("%s%s", c.Settings.Endpoint, uri), request, response)
}

func (c *RestClient) DELETE(uri string, request, response interface{}) (int, error) {
	return c.request("DELETE", fmt.Sprintf("%s%s", c.Settings.Endpoint, uri), request, response)
}

func (c *RestClient) request(method, url string, request, response interface{}) (int, error) {
	var req *http.Request

	if request != nil {
		p, err := json.Marshal(&request)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		req, err = http.NewRequest(method, url, bytes.NewBuffer(p))
		if err != nil {
			return http.StatusBadRequest, err
		}
	} else {
		var err error
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return http.StatusBadRequest, err
		}
	}

	return c.roundTrip(req, response)
}

func (c *RestClient) roundTrip(req *http.Request, response interface{}) (int, error) {

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", c.Settings.UserAgent) // FIXME port this to apikit
	if c.Settings.Credentials.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Settings.Credentials.Token)
	}
	if c.Trace != "" {
		req.Header.Set("X-Request-ID", c.Trace)
		req.Header.Set("X-Force-Trace", c.Trace)
	}

	// perform the request
	resp, err := c.HttpClient.Transport.RoundTrip(req)
	if err != nil {
		if resp == nil {
			return http.StatusInternalServerError, err
		}
		return resp.StatusCode, err
	}

	defer resp.Body.Close()

	// anything other than OK, Created, Accepted, NoContent is treated as an error
	if resp.StatusCode > http.StatusNoContent {
		return resp.StatusCode, ErrApiInvocationError
	}

	// unmarshal the response if one is expected
	if response != nil {
		err = json.NewDecoder(resp.Body).Decode(response)
		if err != nil {
			fmt.Println(err)
			return http.StatusInternalServerError, err
		}
	}

	return resp.StatusCode, nil
}

func NewTransport(logger logger.Logger, transport http.RoundTripper) *http.Client {
	retryTransport := rehttp.NewTransport(
		transport,
		rehttp.RetryAll(
			rehttp.RetryMaxRetries(3),
			rehttp.RetryAny(
				rehttp.RetryTemporaryErr(),
				rehttp.RetryStatuses(502, 503),
			),
		),
		rehttp.ExpJitterDelay(100*time.Millisecond, 1*time.Second),
	)

	return &http.Client{
		Transport: &loggingTransport{
			InnerTransport: retryTransport,
			Logger:         logger,
		},
	}
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := context.WithValue(req.Context(), contextKeyRequestStart, time.Now())
	req = req.WithContext(ctx)

	t.logRequest(req)

	resp, err := t.InnerTransport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	t.logResponse(resp)

	return resp, err
}

func (t *loggingTransport) logRequest(req *http.Request) {

	t.Logger.Debugf("--> %s %s\n", req.Method, req.URL)

	if req.Body == nil {
		return
	}

	defer req.Body.Close()

	data, err := io.ReadAll(req.Body)

	if err != nil {
		t.Logger.Debug("error reading request body:", err)
	} else {
		t.Logger.Debug(string(data))
	}

	if req.Body != nil {
		t.Logger.Debug(req.Body)
	}

	req.Body = io.NopCloser(bytes.NewReader(data))
}

func (t *loggingTransport) logResponse(resp *http.Response) {
	ctx := resp.Request.Context()
	defer resp.Body.Close()

	if start, ok := ctx.Value(contextKeyRequestStart).(time.Time); ok {
		t.Logger.Debugf("<-- %d %s (%s)\n", resp.StatusCode, resp.Request.URL, Duration(time.Since(start), 2))
	} else {
		t.Logger.Debugf("<-- %d %s\n", resp.StatusCode, resp.Request.URL)
	}

	data, err := io.ReadAll(resp.Body)

	if err != nil {
		t.Logger.Debug("error reading response body:", err)
	} else {
		t.Logger.Debug(string(data))
	}

	resp.Body = io.NopCloser(bytes.NewReader(data))
}

// NewStatus initializes a new StatusObject
func NewStatus(s int, m string) StatusObject {
	return StatusObject{Status: s, Message: m}
}

// NewErrorStatus initializes a new StatusObject from an error
func NewErrorStatus(s int, e error, hint string) StatusObject {
	if hint != "" {
		return StatusObject{Status: s, Message: fmt.Sprintf("%s (%s)", e.Error(), hint), RootError: e}
	}
	return StatusObject{Status: s, Message: e.Error(), RootError: e}
}

func (so *StatusObject) String() string {
	return fmt.Sprintf("%s: %d", so.Message, so.Status)
}

func (so *StatusObject) Error() string {
	return so.String()
}
