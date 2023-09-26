package schema_test

import (
	"strings"
	"testing"

	"github.com/reddec/web-form/internal/schema"
	"github.com/stretchr/testify/require"
)

func TestPolicy_UnmarshalText(t *testing.T) {
	t.Run("groups", func(t *testing.T) {
		const txt = `
policy: '"admin" in groups'
`

		f, err := schema.FormsFromStream(strings.NewReader(txt))
		require.NoError(t, err)
		require.NotEmpty(t, f)
		form := f[0]
		require.NotNil(t, form.Policy)

		creds := &schema.Credentials{
			User:   "foo",
			Groups: []string{"bar", "admin"},
			Email:  "foo@xample.com",
		}

		require.True(t, form.IsAllowed(creds))

		// check negative
		creds = &schema.Credentials{
			User:   "foo",
			Groups: []string{"bar", "user"},
			Email:  "foo@xample.com",
		}

		require.False(t, form.IsAllowed(creds))
	})

	t.Run("name", func(t *testing.T) {
		const txt = `
policy: 'user == "admin"'
`

		f, err := schema.FormsFromStream(strings.NewReader(txt))
		require.NoError(t, err)
		require.NotEmpty(t, f)
		form := f[0]
		require.NotNil(t, form.Policy)

		creds := &schema.Credentials{
			User:   "admin",
			Groups: []string{"bar", "admin"},
			Email:  "foo@xample.com",
		}

		require.True(t, form.IsAllowed(creds))

		creds = &schema.Credentials{
			User:   "not-admin",
			Groups: []string{"bar", "admin"},
			Email:  "foo@xample.com",
		}

		require.False(t, form.IsAllowed(creds))
	})

	t.Run("emain", func(t *testing.T) {
		const txt = `
policy: 'email.endsWith("@reddec.net")'
`

		f, err := schema.FormsFromStream(strings.NewReader(txt))
		require.NoError(t, err)
		require.NotEmpty(t, f)
		form := f[0]
		require.NotNil(t, form.Policy)

		creds := &schema.Credentials{
			User:   "admin",
			Groups: []string{"bar", "admin"},
			Email:  "foo@reddec.net",
		}

		require.True(t, form.IsAllowed(creds))

		creds = &schema.Credentials{
			User:   "not-admin",
			Groups: []string{"bar", "admin"},
			Email:  "foo@xample.com",
		}

		require.False(t, form.IsAllowed(creds))
	})

	t.Run("incorrect policy", func(t *testing.T) {
		const txt = `
policy: '"admin" ins groups'
`

		_, err := schema.FormsFromStream(strings.NewReader(txt))
		require.Error(t, err)
	})

	t.Run("type cast always false", func(t *testing.T) {
		const txt = `
policy: 1
`

		f, err := schema.FormsFromStream(strings.NewReader(txt))
		require.NoError(t, err)
		require.NotEmpty(t, f)
		form := f[0]
		require.NotNil(t, form.Policy)

		creds := &schema.Credentials{
			User:   "admin",
			Groups: []string{"bar", "admin"},
			Email:  "foo@reddec.net",
		}

		require.False(t, form.IsAllowed(creds))
	})

	t.Run("nil policy means allowed", func(t *testing.T) {
		const txt = `name: foo`

		f, err := schema.FormsFromStream(strings.NewReader(txt))
		require.NoError(t, err)
		require.NotEmpty(t, f)
		form := f[0]
		require.Nil(t, form.Policy)

		creds := &schema.Credentials{
			User:   "admin",
			Groups: []string{"bar", "admin"},
			Email:  "foo@reddec.net",
		}

		require.True(t, form.IsAllowed(creds))
	})
}
