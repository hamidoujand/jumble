package auth

import (
	"context"
	"errors"

	"github.com/hamidoujand/jumble/internal/domains/user/bus"
)

type key int

const claimsKey key = 1
const userKey key = 2

func SetClaims(ctx context.Context, c Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func GetClaims(ctx context.Context) (Claims, error) {
	c, ok := ctx.Value(claimsKey).(Claims)
	if !ok {
		return Claims{}, errors.New("claims not found in context")
	}

	return c, nil
}

func SetUser(ctx context.Context, usr bus.User) context.Context {
	return context.WithValue(ctx, userKey, usr)
}

func GetUser(ctx context.Context) (bus.User, error) {
	usr, ok := ctx.Value(userKey).(bus.User)
	if !ok {
		return bus.User{}, errors.New("user not found in context")
	}

	return usr, nil
}
