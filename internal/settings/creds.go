// Most of the code is lifted from
// https://github.com/googleapis/google-api-go-client/blob/main/internal/settings.go
//
// For details and copyright etc. see above url.
package settings

import (
	"strings"

	"github.com/txsvc/stdlib/v2"
)

const (
	ClientID     = "CLIENT_ID"
	ClientSecret = "CLIENT_SECRET"
	AccessToken  = "ACCESS_TOKEN"
)

type (
	Credentials struct {
		ProjectID string `json:"project_id,omitempty"` // may be empty
		UserID    string `json:"user_id,omitempty"`    // may be empty, aka client_id
		Token     string `json:"token,omitempty"`      // may be empty
		Expires   int64  `json:"expires,omitempty"`    // 0 = never, > 0 = unix timestamp, < 0 = invalid
	}
)

func CredentialsFromEnv() *Credentials {

	c := &Credentials{
		Token: stdlib.GetString(AccessToken, ""),
	}
	if c.Token == "" {
		c.UserID = stdlib.GetString(ClientID, "")
		c.Token = stdlib.GetString(ClientSecret, "")
	}

	return c
}

// Clone returns a deep copy of the credentials
func (c *Credentials) Clone() *Credentials {
	return &Credentials{
		ProjectID: c.ProjectID,
		UserID:    c.UserID,
		Token:     c.Token,
		Expires:   c.Expires,
	}
}

func (c *Credentials) Key() string {
	return strings.ToLower(c.ProjectID + "." + c.UserID) // FIXME: make it a md5 ?
}

// IsValid test if Crendentials is valid
// FIXME
func (c *Credentials) IsValid() bool {
	if c.Expires < 0 || len(c.Token) == 0 || len(c.ProjectID) == 0 || len(c.UserID) == 0 {
		return false
	}
	return !c.Expired()
}

// Expired only verifies just that, does not check all other attributes
func (c *Credentials) Expired() bool {
	if c.Expires == 0 {
		return false
	}
	return c.Expires < stdlib.Now()
}
