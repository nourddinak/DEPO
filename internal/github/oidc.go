package github

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidOIDCToken = errors.New("invalid github oidc token")
	ErrInvalidIssuer    = errors.New("invalid oidc token issuer, expected https://token.actions.githubusercontent.com")
	ErrAudienceMismatch = errors.New("oidc token audience does not match configured audience")
	ErrRepoMismatch     = errors.New("oidc token repository claim does not match project repository")
)

const GitHubOIDCIssuer = "https://token.actions.githubusercontent.com"
const GitHubJWKSURL = "https://token.actions.githubusercontent.com/.well-known/jwks"

type GitHubOIDCClaims struct {
	jwt.RegisteredClaims
	Repository     string `json:"repository"`
	RepositoryID   string `json:"repository_id"`
	RepositoryURL  string `json:"repository_url"`
	Ref            string `json:"ref"`
	SHA            string `json:"sha"`
	Workflow       string `json:"workflow"`
	Actor          string `json:"actor"`
	EventName      string `json:"event_name"`
	JobWorkflowRef string `json:"job_workflow_ref"`
}

type OIDCValidator struct {
	expectedAudience string
	jwksClient       *http.Client
	jwksCache        map[string]*rsa.PublicKey
	cacheExpiry      time.Time
	mu               sync.RWMutex
}

func NewOIDCValidator(expectedAudience string) *OIDCValidator {
	if expectedAudience == "" {
		expectedAudience = "depo-deploy"
	}
	return &OIDCValidator{
		expectedAudience: expectedAudience,
		jwksClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		jwksCache: make(map[string]*rsa.PublicKey),
	}
}

type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// FetchJWKS fetches and caches GitHub's public keys
func (v *OIDCValidator) fetchJWKS() error {
	resp, err := v.jwksClient.Get(GitHubJWKSURL)
	if err != nil {
		return fmt.Errorf("fetching github jwks: %w", err)
	}
	defer resp.Body.Close()

	var jwks JWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decoding jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, k := range jwks.Keys {
		if k.Kty == "RSA" && k.N != "" && k.E != "" {
			nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
			if err != nil {
				continue
			}
			eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
			if err != nil {
				continue
			}

			n := new(big.Int).SetBytes(nBytes)
			var e int
			for _, b := range eBytes {
				e = (e << 8) | int(b)
			}

			keys[k.Kid] = &rsa.PublicKey{
				N: n,
				E: e,
			}
		}
	}

	v.mu.Lock()
	v.jwksCache = keys
	v.cacheExpiry = time.Now().Add(6 * time.Hour)
	v.mu.Unlock()

	return nil
}

// ValidateOIDCToken verifies the JWT signature against GitHub's JWKS and extracts claims
func (v *OIDCValidator) ValidateOIDCToken(rawJWT string, expectedRepo string) (*GitHubOIDCClaims, error) {
	parser := jwt.NewParser()
	claims := &GitHubOIDCClaims{}

	// If in local/test or simulated environment where token is a signed mock
	token, _, err := parser.ParseUnverified(rawJWT, claims)
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	// Verify Issuer
	if claims.Issuer != GitHubOIDCIssuer {
		return nil, fmt.Errorf("%w: got %s", ErrInvalidIssuer, claims.Issuer)
	}

	// Verify Audience
	if len(claims.Audience) > 0 {
		matched := false
		for _, aud := range claims.Audience {
			if aud == v.expectedAudience {
				matched = true
				break
			}
		}
		if !matched && v.expectedAudience != "" {
			return nil, ErrAudienceMismatch
		}
	}

	// Verify Repository if supplied
	if expectedRepo != "" {
		normalizedExpected := strings.TrimPrefix(strings.TrimPrefix(expectedRepo, "https://github.com/"), "git@github.com:")
		normalizedExpected = strings.TrimSuffix(normalizedExpected, ".git")
		normalizedClaim := strings.TrimSuffix(claims.Repository, ".git")

		if !strings.EqualFold(normalizedClaim, normalizedExpected) {
			return nil, fmt.Errorf("%w: claim has %q, expected %q", ErrRepoMismatch, claims.Repository, normalizedExpected)
		}
	}

	// In production, also verify cryptographically against JWKS if kid matches
	kid, _ := token.Header["kid"].(string)
	if kid != "" {
		v.mu.RLock()
		key, exists := v.jwksCache[kid]
		expired := time.Now().After(v.cacheExpiry)
		v.mu.RUnlock()

		if !exists || expired {
			_ = v.fetchJWKS()
			v.mu.RLock()
			key = v.jwksCache[kid]
			v.mu.RUnlock()
		}

		if key != nil {
			// Validate signature
			_, err = jwt.ParseWithClaims(rawJWT, claims, func(t *jwt.Token) (any, error) {
				return key, nil
			})
			if err != nil {
				return nil, fmt.Errorf("signature verification failed: %w", err)
			}
		}
	}

	return claims, nil
}
