package internal

import (
	"github.com/redhat-partner-ecosystem/shadowcar/internal/settings"
)

type ClientOption interface {
	Apply(ds *settings.DialSettings)
}

// WithEndpoint returns a ClientOption that overrides the default endpoint to be used for a service.
func WithEndpoint(url string) ClientOption {
	return withEndpoint(url)
}

type withEndpoint string

func (w withEndpoint) Apply(ds *settings.DialSettings) {
	ds.Endpoint = string(w)
}

// WithCredentials returns a ClientOption that overrides the default credentials used for a service.
func WithCredentials(userid, token string) ClientOption {
	return withCredentials{
		userID: userid,
		token:  token,
	}
}

type withCredentials struct {
	userID string
	token  string
}

func (w withCredentials) Apply(ds *settings.DialSettings) {
	if ds.Credentials == nil {
		ds.Credentials = &settings.Credentials{}
	}
	ds.Credentials.UserID = w.userID
	ds.Credentials.Token = w.token
}
