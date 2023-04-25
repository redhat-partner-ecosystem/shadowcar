// Most of the code is lifted from
// https://github.com/googleapis/google-api-go-client/blob/main/internal/settings.go
//
// For details and copyright etc. see above url.
package settings

type (
	State int

	// DialSettings holds information needed to establish a connection with a
	// backend API service or to simply configure a service/CLI.
	DialSettings struct {
		Endpoint string `json:"endpoint,omitempty"`

		Credentials *Credentials `json:"credentials,omitempty"`

		Scopes        []string `json:"scopes,omitempty"`
		DefaultScopes []string `json:"default_scopes,omitempty"`

		UserAgent string `json:"user_agent,omitempty"`

		Options map[string]string `json:"options,omitempty"` // holds all other values ...
	}
)

func (ds *DialSettings) Clone() DialSettings {
	s := DialSettings{
		Endpoint:  ds.Endpoint,
		UserAgent: ds.UserAgent,
	}

	if len(ds.Scopes) > 0 {
		s.Scopes = make([]string, len(ds.Scopes))
		copy(s.Scopes, ds.Scopes)
	}
	if len(ds.DefaultScopes) > 0 {
		s.DefaultScopes = make([]string, len(ds.DefaultScopes))
		copy(s.DefaultScopes, ds.DefaultScopes)
	}

	if ds.Credentials != nil {
		s.Credentials = ds.Credentials.Clone()
	}
	if len(ds.Options) > 0 {
		s.Options = make(map[string]string)
		for k, v := range ds.Options {
			s.Options[k] = v
		}
	}
	return s
}

// GetScopes returns the user-provided scopes, if set, or else falls back to the default scopes.
func (ds *DialSettings) GetScopes() []string {
	if len(ds.Scopes) > 0 {
		return ds.Scopes
	}
	return ds.DefaultScopes
}

// HasOption returns true if ds has a custom option opt.
func (ds *DialSettings) HasOption(opt string) bool {
	_, ok := ds.Options[opt]
	return ok
}

// GetOption returns the custom option opt if it exists or an empty string otherwise
func (ds *DialSettings) GetOption(opt string) string {
	if o, ok := ds.Options[opt]; ok {
		return o
	}
	return ""
}

// SetOptions registers a custom option o with key opt.
func (ds *DialSettings) SetOption(opt, o string) {
	if ds.Options == nil {
		ds.Options = make(map[string]string)
	}
	ds.Options[opt] = o
}
