package oauth2

import (
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestDecodeScopesMatcherHookFunc(t *testing.T) {
	t.Parallel()

	type Type struct {
		Matcher ScopesMatcher `mapstructure:"scopes"`
	}

	decode := func(data []byte) map[any]any {
		var res map[any]any

		err := yaml.Unmarshal(data, &res)
		require.NoError(t, err)

		return res
	}

	for _, tc := range []struct {
		uc     string
		config []byte
		assert func(t *testing.T, err error, matcher *ScopesMatcher)
	}{
		{
			uc: "structure with scopes under value and wildcard strategy",
			config: []byte(`
scopes:
  values:
    - foo
    - bar
  matching_strategy: wildcard
`),
			assert: func(t *testing.T, err error, matcher *ScopesMatcher) {
				t.Helper()

				require.NoError(t, err)

				assert.IsType(t, WildcardScopeStrategyMatcher{}, matcher.Matcher)
				assert.ElementsMatch(t, matcher.Scopes, []string{"foo", "bar"})
			},
		},
		{
			uc: "structure with scopes under value and exact strategy",
			config: []byte(`
scopes:
  values:
    - foo
    - bar
  matching_strategy: exact
`),
			assert: func(t *testing.T, err error, matcher *ScopesMatcher) {
				t.Helper()

				require.NoError(t, err)

				assert.IsType(t, ExactScopeStrategyMatcher{}, matcher.Matcher)
				assert.ElementsMatch(t, matcher.Scopes, []string{"foo", "bar"})
			},
		},
		{
			uc: "structure with scopes under value and hierarchic strategy",
			config: []byte(`
scopes:
  values:
    - foo
    - bar
  matching_strategy: hierarchic
`),
			assert: func(t *testing.T, err error, matcher *ScopesMatcher) {
				t.Helper()

				require.NoError(t, err)

				assert.IsType(t, HierarchicScopeStrategyMatcher{}, matcher.Matcher)
				assert.ElementsMatch(t, matcher.Scopes, []string{"foo", "bar"})
			},
		},
		{
			uc: "only scopes provided under values property",
			config: []byte(`
scopes:
  values:
    - foo
    - bar
`),
			assert: func(t *testing.T, err error, matcher *ScopesMatcher) {
				t.Helper()

				require.NoError(t, err)

				assert.IsType(t, ExactScopeStrategyMatcher{}, matcher.Matcher)
				assert.ElementsMatch(t, matcher.Scopes, []string{"foo", "bar"})
			},
		},
		{
			uc: "only scopes provided on top level",
			config: []byte(`
scopes:
  - foo
  - bar
`),
			assert: func(t *testing.T, err error, matcher *ScopesMatcher) {
				t.Helper()

				require.NoError(t, err)

				assert.IsType(t, ExactScopeStrategyMatcher{}, matcher.Matcher)
				assert.ElementsMatch(t, matcher.Scopes, []string{"foo", "bar"})
			},
		},
		{
			uc: "no scopes provided, but matching strategy",
			config: []byte(`
scopes:
  matching_strategy: exact
`),
			assert: func(t *testing.T, err error, matcher *ScopesMatcher) {
				t.Helper()

				assert.Error(t, err)
			},
		},
		{
			uc: "malformed configuration",
			config: []byte(`
scopes:
  foo: bar
`),
			assert: func(t *testing.T, err error, matcher *ScopesMatcher) {
				t.Helper()

				assert.Error(t, err)
			},
		},
	} {
		t.Run("case="+tc.uc, func(t *testing.T) {
			// GIVEN
			var typ Type

			dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				DecodeHook: mapstructure.ComposeDecodeHookFunc(
					DecodeScopesMatcherHookFunc(),
				),
				Result: &typ,
			})
			require.NoError(t, err)

			// WHEN
			err = dec.Decode(decode(tc.config))

			// THEN
			tc.assert(t, err, &typ.Matcher)
		})
	}
}
