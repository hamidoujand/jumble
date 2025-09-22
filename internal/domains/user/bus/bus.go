package bus

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/jumble/internal/page"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicatedEmail = errors.New("email already in use")
	ErrUserNotFound    = errors.New("user not found")
)

type store interface {
	Create(ctx context.Context, usr User) error
	Update(ctx context.Context, usr User) error
	Delete(ctx context.Context, usr User) error
	QueryByID(ctx context.Context, userId uuid.UUID) (User, error)
	QueryByEmail(ctx context.Context, email mail.Address) (User, error)
	Query(ctx context.Context, filters QueryFilter, orderBy Field, page page.Page) ([]User, error)
	Count(ctx context.Context, filters QueryFilter) (int, error)
}

type Bus struct {
	store store
}

func New(store store) *Bus {
	return &Bus{store: store}
}

func (b *Bus) Create(ctx context.Context, nu NewUser) (User, error) {
	bs, err := bcrypt.GenerateFromPassword([]byte(nu.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("generateFromPassword: %w", err)
	}

	//strip the monotonic clock from the time to not mess your timestamps.
	//removes the nanoseconds and keeps the microseconds.
	now := time.Now().Truncate(time.Microsecond)

	usr := User{
		ID:           uuid.New(),
		Name:         nu.Name,
		Email:        nu.Email,
		Roles:        nu.Roles,
		PasswordHash: bs,
		Department:   nu.Department,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := b.store.Create(ctx, usr); err != nil {
		return User{}, fmt.Errorf("create: %w", err)
	}

	return usr, nil
}

func (b *Bus) Update(ctx context.Context, usr User, updates UpdateUser) (User, error) {
	if updates.Name != nil {
		usr.Name = *updates.Name
	}

	if updates.Email != nil {
		usr.Email = *updates.Email
	}

	if updates.Department != nil {
		usr.Department = *updates.Department
	}

	if updates.Roles != nil {
		usr.Roles = updates.Roles
	}

	if updates.Password != nil {
		bs, err := bcrypt.GenerateFromPassword([]byte(*updates.Password), bcrypt.DefaultCost)
		if err != nil {
			return User{}, fmt.Errorf("generateFromPassword: %w", err)
		}

		usr.PasswordHash = bs
	}

	if updates.Enabled != nil {
		usr.Enabled = *updates.Enabled
	}

	usr.UpdatedAt = time.Now()
	if err := b.store.Update(ctx, usr); err != nil {
		return User{}, fmt.Errorf("update: %w", err)
	}

	return usr, nil
}

func (b *Bus) Delete(ctx context.Context, usr User) error {
	if err := b.store.Delete(ctx, usr); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}

func (b *Bus) QueryByID(ctx context.Context, id uuid.UUID) (User, error) {
	usr, err := b.store.QueryByID(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("queryByID: %w", err)
	}

	return usr, nil
}

func (b *Bus) QueryByEmail(ctx context.Context, email mail.Address) (User, error) {
	usr, err := b.store.QueryByEmail(ctx, email)
	if err != nil {
		return User{}, fmt.Errorf("queryByEmail: %w", err)
	}

	return usr, nil
}

func (b *Bus) Count(ctx context.Context, filter QueryFilter) (int, error) {
	return b.store.Count(ctx, filter)
}

func (b *Bus) Query(ctx context.Context, filters QueryFilter, order Field, page page.Page) ([]User, error) {
	usrs, err := b.store.Query(ctx, filters, order, page)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	return usrs, nil
}

func (b *Bus) Authenticate(ctx context.Context, email mail.Address, password string) (User, error) {
	usr, err := b.store.QueryByEmail(ctx, email)
	if err != nil {
		return User{}, fmt.Errorf("queryByEmail: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword(usr.PasswordHash, []byte(password)); err != nil {
		return User{}, fmt.Errorf("compareHashAndPass: %w", err)
	}

	return usr, nil
}
