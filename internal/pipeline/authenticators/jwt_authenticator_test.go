package authenticators

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gopkg.in/yaml.v2"

	"github.com/dadrus/heimdall/internal/pipeline/authenticators/extractors"
	"github.com/dadrus/heimdall/internal/pipeline/endpoint"
	"github.com/dadrus/heimdall/internal/pipeline/oauth2"
	"github.com/dadrus/heimdall/internal/pipeline/subject"
	"github.com/dadrus/heimdall/internal/pipeline/testsupport"
)

func TestCreateJwtAuthenticator(t *testing.T) {
	t.Parallel()

	decode := func(data []byte) map[string]interface{} {
		var res map[string]interface{}

		err := yaml.Unmarshal(data, &res)
		assert.NoError(t, err)

		return res
	}

	// nolint
	for _, tc := range []struct {
		uc     string
		config []byte
		assert func(t *testing.T, err error, a *jwtAuthenticator)
	}{
		{
			uc: "missing jwks url config",
			config: []byte(`
jwt_token_from:
  - header: foo-header
jwt_assertions:
  issuers:
    - foobar
session:
  subject_from: some_template`),
			assert: func(t *testing.T, err error, a *jwtAuthenticator) {
				t.Helper()
				assert.Error(t, err)
			},
		},
		{
			uc: "missing trusted_issuers config",
			config: []byte(`
jwks_endpoint:
  url: http://test.com
jwt_assertions:
  audiences:
    - foobar
session:
  subject_from: some_template`),
			assert: func(t *testing.T, err error, a *jwtAuthenticator) {
				t.Helper()
				assert.Error(t, err)
			},
		},
		{
			uc: "config with undefined fields",
			config: []byte(`
jwks_endpoint:
  url: http://test.com
jwt_token_from:
  - header: foo-header
jwt_assertions:
  issuers:
    - foobar
foo: bar`),
			assert: func(t *testing.T, err error, a *jwtAuthenticator) {
				t.Helper()
				assert.Error(t, err)
			},
		},
		{
			uc: "valid configuration with defaults",
			config: []byte(`
jwks_endpoint:
  url: http://test.com
jwt_assertions:
  issuers:
    - foobar`),
			assert: func(t *testing.T, err error, auth *jwtAuthenticator) {
				t.Helper()
				require.NoError(t, err)

				// endpoint settings
				assert.Equal(t, "http://test.com", auth.e.URL)
				assert.Equal(t, "GET", auth.e.Method)
				assert.Equal(t, 1, len(auth.e.Headers))
				assert.Contains(t, auth.e.Headers, "Accept-Type")
				assert.Equal(t, auth.e.Headers["Accept-Type"], "application/json")

				// token extractor settings
				assert.IsType(t, extractors.CompositeExtractStrategy{}, auth.adg)
				assert.Contains(t, auth.adg, extractors.HeaderValueExtractStrategy{Name: "Authorization", Prefix: "Bearer"})
				assert.Contains(t, auth.adg, extractors.CookieValueExtractStrategy{Name: "access_token"})
				assert.Contains(t, auth.adg, extractors.QueryParameterExtractStrategy{Name: "access_token"})

				// assertions settings
				assert.NoError(t, auth.a.ScopesMatcher.MatchScopes([]string{}))
				assert.Empty(t, auth.a.TargetAudiences)
				assert.Len(t, auth.a.TrustedIssuers, 1)
				assert.Contains(t, auth.a.TrustedIssuers, "foobar")
				assert.Len(t, auth.a.AllowedAlgorithms, 6)

				assert.ElementsMatch(t, auth.a.AllowedAlgorithms, []string{
					string(jose.ES256), string(jose.ES384), string(jose.ES512),
					string(jose.PS256), string(jose.PS384), string(jose.PS512),
				})
				assert.Equal(t, time.Duration(0), auth.a.ValidityLeeway)

				// session settings
				sess, ok := auth.se.(*Session)
				require.True(t, ok)
				assert.Equal(t, "sub", sess.SubjectFrom)
				assert.Empty(t, sess.AttributesFrom)
			},
		},
		{
			uc: "valid configuration with overwrites",
			config: []byte(`
jwks_endpoint:
  url: http://test.com
  method: POST
  headers:
    Accept-Type: application/foobar
jwt_token_from:
  - header: foo-header
jwt_assertions:
  scopes:
    matching_strategy: wildcard
    values:
      - foo
  issuers:
    - foobar
  allowed_algorithms:
    - ES256
session:
  subject_from: some_claim`),
			assert: func(t *testing.T, err error, auth *jwtAuthenticator) {
				t.Helper()
				require.NoError(t, err)

				// endpoint settings
				assert.Equal(t, "http://test.com", auth.e.URL)
				assert.Equal(t, "POST", auth.e.Method)
				assert.Equal(t, 1, len(auth.e.Headers))
				assert.Contains(t, auth.e.Headers, "Accept-Type")
				assert.Equal(t, auth.e.Headers["Accept-Type"], "application/foobar")

				// token extractor settings
				assert.IsType(t, extractors.CompositeExtractStrategy{}, auth.adg)
				assert.Len(t, auth.adg, 1)
				assert.Contains(t, auth.adg, &extractors.HeaderValueExtractStrategy{Name: "foo-header"})

				// assertions settings
				assert.NotNil(t, auth.a.ScopesMatcher)
				assert.NoError(t, auth.a.ScopesMatcher.MatchScopes([]string{"foo"}))
				assert.Empty(t, auth.a.TargetAudiences)
				assert.Len(t, auth.a.TrustedIssuers, 1)
				assert.Contains(t, auth.a.TrustedIssuers, "foobar")
				assert.Len(t, auth.a.AllowedAlgorithms, 1)

				assert.ElementsMatch(t, auth.a.AllowedAlgorithms, []string{string(jose.ES256)})
				assert.Equal(t, time.Duration(0), auth.a.ValidityLeeway)

				// session settings
				sess, ok := auth.se.(*Session)
				require.True(t, ok)
				assert.Equal(t, "some_claim", sess.SubjectFrom)
				assert.Empty(t, sess.AttributesFrom)
			},
		},
	} {
		t.Run("case="+tc.uc, func(t *testing.T) {
			t.Parallel()

			// WHEN
			a, err := newJwtAuthenticator(decode(tc.config))

			// THEN
			tc.assert(t, err, a)
		})
	}
}

func setup(t *testing.T, subject, issuer, audience string) ([]byte, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	jwks := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{Key: privateKey.Public(), KeyID: "bar", Algorithm: string(jose.PS512)},
		},
	}
	rawJwks, err := json.Marshal(jwks)
	require.NoError(t, err)

	signerOpts := jose.SignerOptions{}
	signerOpts.WithType("JWT").WithHeader("kid", "bar")
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.PS512, Key: privateKey}, &signerOpts)
	require.NoError(t, err)

	builder := jwt.Signed(signer)
	builder = builder.Claims(map[string]interface{}{
		"sub": subject,
		"iss": issuer,
		"jti": "foo",
		"iat": time.Now().Unix() - 1,
		"nbf": time.Now().Unix() - 1,
		"exp": time.Now().Unix() + 2,
		"aud": []string{audience},
		"scp": []string{"foo", "bar"},
	})
	rawJwt, err := builder.CompactSerialize()
	require.NoError(t, err)

	return rawJwks, rawJwt
}

func TestCreateJwtAuthenticatorFromPrototype(t *testing.T) {
	t.Parallel()

	// GIVEN
	prototypeConfig, err := testsupport.DecodeTestConfig([]byte(`
jwks_endpoint:
  url: http://test.com
jwt_assertions:
  issuers:
    - foobar`))
	require.NoError(t, err)

	val := []byte(`
jwt_assertions:
  issuers:
    - barfoo
  allowed_algorithms:
    - ES512`)

	config, err := testsupport.DecodeTestConfig(val)
	require.NoError(t, err)

	prototype, err := newJwtAuthenticator(prototypeConfig)
	require.NoError(t, err)

	// WHEN
	auth, err := prototype.WithConfig(config)

	// THEN
	require.NoError(t, err)

	jwta, ok := auth.(*jwtAuthenticator)
	require.True(t, ok)
	assert.Equal(t, prototype.e, jwta.e)
	assert.Equal(t, prototype.adg, jwta.adg)
	assert.Equal(t, prototype.se, jwta.se)
	assert.NotEqual(t, prototype.a, jwta.a)

	assert.NoError(t, jwta.a.ScopesMatcher.MatchScopes([]string{}))
	assert.Empty(t, jwta.a.TargetAudiences)
	assert.ElementsMatch(t, jwta.a.TrustedIssuers, []string{"barfoo"})
	assert.ElementsMatch(t, jwta.a.AllowedAlgorithms, []string{string(jose.ES512)})
}

func TestSuccessfulExecutionOfJwtAuthenticator(t *testing.T) {
	t.Parallel()

	// GIVEN
	var receivedAcceptType string

	subjectID := "foo"
	issuer := "foobar"
	audience := "bar"
	jwks, jwt := setup(t, subjectID, issuer, audience)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}

		receivedAcceptType = r.Header.Get("Accept-Type")

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(jwks)))

		_, err := w.Write(jwks)
		assert.NoError(t, err)
	}))
	defer srv.Close()

	as := oauth2.Expectation{
		ScopesMatcher: oauth2.ScopesMatcher{
			Match:  oauth2.ExactScopeStrategy,
			Scopes: []string{"foo"},
		},
		TargetAudiences:   []string{audience},
		TrustedIssuers:    []string{issuer},
		AllowedAlgorithms: []string{string(jose.PS512)},
		ValidityLeeway:    1 * time.Minute,
	}

	ctx := &testsupport.MockContext{}
	ctx.On("AppContext").Return(context.Background())

	adg := &mockAuthDataGetter{}
	adg.On("GetAuthData", ctx).Return(dummyAuthData{Val: jwt}, nil)

	encJwtPayload := strings.Split(jwt, ".")[1]
	rawPaload, err := base64.RawStdEncoding.DecodeString(encJwtPayload)
	require.NoError(t, err)

	var attrs map[string]any
	err = json.Unmarshal(rawPaload, &attrs)
	require.NoError(t, err)

	se := &testsupport.MockSubjectExtractor{}
	se.On("GetSubject", rawPaload).Return(&subject.Subject{ID: subjectID, Attributes: attrs}, nil)

	auth := jwtAuthenticator{
		e: endpoint.Endpoint{
			URL:     srv.URL,
			Method:  http.MethodGet,
			Headers: map[string]string{"Accept-Type": "application/json"},
		},
		a:   as,
		se:  se,
		adg: adg,
	}

	// WHEN
	sub, err := auth.Authenticate(ctx)

	// THEN
	require.NoError(t, err)

	assert.NotNil(t, sub)
	assert.Equal(t, subjectID, sub.ID)
	assert.Equal(t, attrs, sub.Attributes)
	assert.Equal(t, "application/json", receivedAcceptType)

	ctx.AssertExpectations(t)
	se.AssertExpectations(t)
	adg.AssertExpectations(t)
}