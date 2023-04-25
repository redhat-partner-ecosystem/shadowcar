package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScopes(t *testing.T) {
	cfg1 := DialSettings{
		DefaultScopes: []string{"a", "b", "c"},
	}
	assert.NotEmpty(t, cfg1.GetScopes())

	cfg2 := DialSettings{
		Scopes: []string{"A", "B"},
	}
	assert.NotEmpty(t, cfg2.GetScopes())

	cfg3 := DialSettings{
		Scopes:        []string{"A", "B"},
		DefaultScopes: []string{"a", "b", "c"},
	}
	assert.NotEmpty(t, cfg3.GetScopes())
	assert.Equal(t, []string{"A", "B"}, cfg3.GetScopes())
}

func TestOptions(t *testing.T) {
	cfg1 := DialSettings{}
	assert.Nil(t, cfg1.Options)
	assert.False(t, cfg1.HasOption("FOO"))

	opt := cfg1.GetOption("FOO")
	assert.Empty(t, opt)

	cfg1.SetOption("FOO", "x")
	assert.True(t, cfg1.HasOption("FOO"))
	opt = cfg1.GetOption("FOO")
	assert.Equal(t, "x", opt)
}

func TestCloneDialSettings(t *testing.T) {
	s1 := DialSettings{
		Endpoint:  "ep",
		UserAgent: "UserAgent",
	}
	dup1 := s1.Clone()
	assert.Equal(t, s1, dup1)

	// adding Scopes
	s1.Scopes = []string{"A", "B"}
	s1.DefaultScopes = []string{"a", "b"}

	dup2 := s1.Clone()
	assert.Equal(t, s1, dup2)

	// adding credentials
	s1.Credentials = &Credentials{
		ProjectID: "p",
		UserID:    "u",
		Token:     "t",
		Expires:   10,
	}

	dup3 := s1.Clone()
	assert.Equal(t, s1, dup3)

	// adding options
	s1.SetOption("foo", "bar")

	dup4 := s1.Clone()
	assert.Equal(t, s1, dup4)
}
