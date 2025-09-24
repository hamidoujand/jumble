package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hamidoujand/jumble/internal/auth"
	"github.com/hamidoujand/jumble/internal/domains/user/bus"
)

func Test_Auth(t *testing.T) {
	ks := newKeyStore(t)

	a := auth.New(ks, nil, "auth_test")

	ts := tests{
		a:  a,
		ks: ks,
	}

	t.Run("generate_token", ts.generateToken)
	t.Run("authorization", ts.authorization)
}

type tests struct {
	a  *auth.Auth
	ks *keyStore
}

func (ts tests) generateToken(t *testing.T) {
	usrId := uuid.NewString()

	c := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "auth_test",
			Subject:   usrId,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Roles: []string{bus.RoleUser.String()},
	}

	token, err := ts.a.GenerateToken(ts.ks.kid, c)
	if err != nil {
		t.Fatalf("failed to generate token: %s", err)
	}

	//validate the token
	bearer := "Bearer " + token
	verifiedClaims, err := ts.a.VerifyToken(context.Background(), bearer)
	if err != nil {
		t.Fatalf("verifyToken: %s", err)
	}

	diff := cmp.Diff(verifiedClaims, c)
	if diff != "" {
		t.Fatalf("claims not match:\n%s\n", diff)
	}
}

func (ts tests) authorization(t *testing.T) {
	usrId := uuid.NewString()

	c := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "auth_test",
			Subject:   usrId,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Roles: []string{bus.RoleAdmin.String()},
	}

	//allow only admins
	roleSet := map[string]struct{}{
		bus.RoleAdmin.String(): {},
	}

	err := ts.a.Authorized(c, roleSet)
	if err != nil {
		t.Fatalf("should pass the authorization: %s", err)
	}

	roleSet = map[string]struct{}{
		bus.RoleUser.String(): {},
	}

	err = ts.a.Authorized(c, roleSet)
	if err == nil {
		t.Fatalf("should not pass the authorization: %s", err)
	}

	if !errors.Is(err, auth.ErrForbidden) {
		t.Errorf("error=%v, got=%v", auth.ErrForbidden, err)
	}
}

// =============================================================================

type keyStore struct {
	pv  *rsa.PrivateKey
	pb  *rsa.PublicKey
	kid string
}

func newKeyStore(t *testing.T) *keyStore {
	pv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generateKey: %s", err)
	}

	return &keyStore{
		pv:  pv,
		pb:  &pv.PublicKey,
		kid: uuid.NewString(),
	}
}

func (ks *keyStore) PrivateKey(kid string) (*rsa.PrivateKey, error) {
	return ks.pv, nil
}
func (ks *keyStore) PublicKey(kid string) (*rsa.PublicKey, error) {
	return ks.pb, nil
}
