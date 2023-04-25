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
	"github.com/rs/zerolog/log"

	"github.com/redhat-partner-ecosystem/shadowcar/internal/settings"
)

const (
	// format error messages
	MsgStatus = "%s. status: %d"
)

// RestClient - API client encapsulating the http client
type (
	RestClient struct {
		HttpClient *http.Client
		Settings   *settings.DialSettings
		Trace      string
	}

	LoggingTransport struct {
		InnerTransport http.RoundTripper
	}

	contextKey struct {
		name string
	}
)

var (
	// ErrApiInvocationError indicates an error in an API call
	ErrApiInvocationError = errors.New("api invocation error")

	ctxKeyRequestStart = &contextKey{"RequestStart"}
)

/*
func NewRestClient(ds *settings.DialSettings) *RestClient {
	var _ds *settings.DialSettings

	httpClient := NewLoggingTransport(http.DefaultTransport)

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
		Trace:      stdlib.GetString(config.ForceTraceENV, ""),
	}
}
*/

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

	if c.Settings.Credentials.UserID != "" && c.Settings.Credentials.Token != "" {
		req.SetBasicAuth(c.Settings.Credentials.UserID, c.Settings.Credentials.Token)
	} else if c.Settings.Credentials.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Settings.Credentials.Token)
	}
	if c.Trace != "" {
		req.Header.Set("X-Request-ID", XID())    // e.g ch3oncmfosvp07shov90
		req.Header.Set("X-Force-Trace", c.Trace) // a predefined value in order to e.g. grep in logs
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

func NewLoggingTransport(transport http.RoundTripper) *http.Client {
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
		Transport: &LoggingTransport{
			InnerTransport: retryTransport,
		},
	}
}

// RoundTrip logs the request and reply if the log level is debug or trace
func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {

	xreqid := XID()

	if log.Debug().Enabled() {
		req = req.WithContext(context.WithValue(req.Context(), ctxKeyRequestStart, time.Now()))
		t.logRequest(req, xreqid)
	}

	resp, err := t.InnerTransport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if log.Debug().Enabled() {
		t.logResponse(resp, xreqid)
	}

	return resp, err
}

func (t *LoggingTransport) logRequest(req *http.Request, reqid string) {

	if req.Body == nil {
		log.Debug().Str("m", req.Method).Str("r", req.URL.RequestURI()).Str("uid", reqid).Msg("REQ")
		return
	}

	defer req.Body.Close()

	data, err := io.ReadAll(req.Body)

	if err != nil {
		log.Error().Err(err).Str("uid", reqid).Msg(err.Error())
	} else {
		if log.Trace().Enabled() {
			log.Trace().Str("m", req.Method).Str("r", req.URL.RequestURI()).Bytes("body", data).Str("uid", reqid).Msg("REQ")
		} else {
			log.Debug().Str("m", req.Method).Str("r", req.URL.RequestURI()).Str("uid", reqid).Msg("REQ")
		}
	}

	req.Body = io.NopCloser(bytes.NewReader(data))
}

func (t *LoggingTransport) logResponse(resp *http.Response, reqid string) {
	ctx := resp.Request.Context()
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("uid", reqid).Msg(err.Error())
	}

	if start, ok := ctx.Value(ctxKeyRequestStart).(time.Time); ok {
		if log.Trace().Enabled() {
			log.Trace().Str("r", resp.Request.URL.RequestURI()).Int("status", resp.StatusCode).Bytes("body", data).Str("d", fmt.Sprintf("%s", Duration(time.Since(start), 2))).Str("uid", reqid).Msg("RESP")
		} else {
			log.Debug().Str("r", resp.Request.URL.RequestURI()).Int("status", resp.StatusCode).Str("d", fmt.Sprintf("%s", Duration(time.Since(start), 2))).Str("uid", reqid).Msg("RESP")
		}
	} else {
		if log.Trace().Enabled() {
			log.Trace().Str("r", resp.Request.URL.RequestURI()).Int("status", resp.StatusCode).Bytes("body", data).Str("uid", reqid).Msg("RESP")
		} else {
			log.Debug().Str("r", resp.Request.URL.RequestURI()).Int("status", resp.StatusCode).Str("uid", reqid).Msg("RESP")
		}
	}

	resp.Body = io.NopCloser(bytes.NewReader(data))
}
