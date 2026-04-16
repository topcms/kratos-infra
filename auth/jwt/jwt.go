package jwt

import (
	"context"
	"errors"
	"fmt"
	"strings"

	infraauth "github.com/topcms/kratos-infra/middleware/auth"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// Config defines validator options.
//
// Current implementation supports HS256 only.
type Config struct {
	Enabled       bool
	SigningMethod string
	Secret        string
	Issuer        string
	Audience      []string
}

// NewTokenValidator creates a TokenValidator for infra auth middleware.
//
// Returns nil,nil when jwt is disabled.
func NewTokenValidator(c Config) (infraauth.TokenValidator, error) {
	if !c.Enabled {
		return nil, nil
	}

	method := strings.ToUpper(strings.TrimSpace(c.SigningMethod))
	if method == "" {
		method = "HS256"
	}
	if method != "HS256" {
		return nil, fmt.Errorf("unsupported jwt signing method: %s", method)
	}
	if strings.TrimSpace(c.Secret) == "" {
		return nil, errors.New("jwt secret is required when jwt is enabled")
	}

	parser := jwtv5.NewParser(jwtv5.WithValidMethods([]string{method}))
	secret := []byte(c.Secret)
	issuer := strings.TrimSpace(c.Issuer)
	audience := c.Audience

	return func(ctx context.Context, token string) error {
		_ = ctx
		claims := &jwtv5.RegisteredClaims{}
		parsed, err := parser.ParseWithClaims(token, claims, func(t *jwtv5.Token) (any, error) {
			return secret, nil
		})
		if err != nil {
			return err
		}
		if parsed == nil || !parsed.Valid {
			return errors.New("invalid jwt token")
		}

		if issuer != "" && claims.Issuer != issuer {
			return fmt.Errorf("invalid jwt issuer: %s", claims.Issuer)
		}

		if len(audience) > 0 {
			matched := false
			for _, expect := range audience {
				expect = strings.TrimSpace(expect)
				if expect == "" {
					continue
				}
				for _, got := range claims.Audience {
					if got == expect {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				return fmt.Errorf("invalid jwt audience: %v", claims.Audience)
			}
		}
		return nil
	}, nil
}
