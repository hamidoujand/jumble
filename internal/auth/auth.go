package auth

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/hamidoujand/jumble/internal/domains/user/bus"
)

var (
	ErrForbidden    = errors.New("attempted action is not allowed")
	ErrKIDMissing   = errors.New("kid missing from token header")
	ErrKIDMalformed = errors.New("kid in token header is malformed")
	ErrUserDisabled = errors.New("user is disabled")
	ErrInvalidToken = errors.New("invalid token")
)

type Claims struct {
	jwt.RegisteredClaims
	Roles []string `json:"roles"`
}

type keyLoader interface {
	PrivateKey(kid string) (*rsa.PrivateKey, error)
	PublicKey(kid string) (*rsa.PublicKey, error)
}

type userBus interface {
	QueryByID(ctx context.Context, id uuid.UUID) (bus.User, error)
}

type Auth struct {
	keyLoader    keyLoader
	signinMethod jwt.SigningMethod
	parser       *jwt.Parser
}

func New(loader keyLoader, usrBus userBus, issuer string) *Auth {
	return &Auth{
		keyLoader:    loader,
		signinMethod: jwt.GetSigningMethod(jwt.SigningMethodRS256.Name),
		parser:       jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name})),
	}
}

func (a *Auth) GenerateToken(kid string, c Claims) (string, error) {
	t := jwt.NewWithClaims(a.signinMethod, c)

	t.Header["kid"] = kid

	privareKey, err := a.keyLoader.PrivateKey(kid)
	if err != nil {
		return "", fmt.Errorf("privateKey: %w", err)
	}

	token, err := t.SignedString(privareKey)
	if err != nil {
		return "", fmt.Errorf("signedString: %w", err)
	}

	return token, nil
}

func (a *Auth) VerifyToken(ctx context.Context, bearer string) (Claims, error) {
	//check for format "Bearer <TOKEN>"
	if !strings.HasPrefix(bearer, "Bearer ") {
		return Claims{}, errors.New("expected authorization header format: Bearer <token>")
	}

	token := bearer[7:] // get rid of "Bearer "

	var claims Claims
	verfiedToken, err := a.parser.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
		//fetch the public key
		k, ok := t.Header["kid"]
		if !ok {
			return nil, ErrKIDMissing
		}

		kid, ok := k.(string)
		if !ok {
			return nil, ErrKIDMalformed
		}

		//fetch the public key for this kid
		pub, err := a.keyLoader.PublicKey(kid)
		if err != nil {
			return nil, fmt.Errorf("fetching public key for kid[%s]: %w", kid, err)
		}

		return pub, nil
	})

	if err != nil {
		return Claims{}, fmt.Errorf("parseWithClaims: %w", err)
	}

	//validate token
	if !verfiedToken.Valid {
		return Claims{}, ErrInvalidToken
	}

	return claims, nil
}

func (a *Auth) Authorized(c Claims, roleSet map[string]struct{}) error {
	for _, role := range c.Roles {
		_, ok := roleSet[role]
		if ok {
			return nil
		}
	}

	return ErrForbidden
}
