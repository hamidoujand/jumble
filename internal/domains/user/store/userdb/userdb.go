package userdb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/mail"

	"github.com/google/uuid"
	usrBus "github.com/hamidoujand/jumble/internal/domains/user/bus"
	"github.com/hamidoujand/jumble/internal/page"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/trace"
)

const (
	uniqueViolation = "23505"
)

type Store struct {
	db     *sqlx.DB
	tracer trace.Tracer
}

func NewStore(db *sqlx.DB, tracer trace.Tracer) *Store {
	if tracer == nil {
		fmt.Printf("Tracer is nil: %+v\n", tracer)
	}
	return &Store{
		db:     db,
		tracer: tracer,
	}
}

func (s *Store) Create(ctx context.Context, usr usrBus.User) error {
	const q = `
	INSERT INTO users (id,name,email,password_hash,roles,enabled,department,created_at,updated_at) 
	VALUES (:id,:name,:email,:password_hash,:roles,:enabled,:department,:created_at,:updated_at) 
	`

	ctx, span := s.tracer.Start(ctx, "user.store.create")
	defer span.End()

	if _, err := s.db.NamedExecContext(ctx, q, fromBusUser(usr)); err != nil {
		var pgerror *pgconn.PgError
		if errors.As(err, &pgerror) {
			//look for duplicated code
			if pgerror.Code == uniqueViolation {
				return usrBus.ErrDuplicatedEmail
			}
		}
		return fmt.Errorf("namedExecContext: %w", err)
	}

	return nil
}

func (s *Store) Update(ctx context.Context, usr usrBus.User) error {
	const q = `
	UPDATE users 
	SET 
		name = :name, 
		email = :email, 
		password_hash = :password_hash,
		roles = :roles,
		enabled = :enabled,
		department = :department, 
		updated_at = :updated_at
	WHERE 
		id = :id;
	`
	ctx, span := s.tracer.Start(ctx, "user.store.update")
	defer span.End()

	if _, err := s.db.NamedExecContext(ctx, q, fromBusUser(usr)); err != nil {
		var pgerror *pgconn.PgError
		if errors.As(err, &pgerror) {
			if pgerror.Code == uniqueViolation {
				return usrBus.ErrDuplicatedEmail
			}
		}
		return fmt.Errorf("namedExecContext: %w", err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, usr usrBus.User) error {
	//passing a value of User type entirely so this layer does not need to do "not found" check
	const q = `
	DELETE FROM users WHERE id = :id;
	`
	ctx, span := s.tracer.Start(ctx, "user.store.delete")
	defer span.End()

	if _, err := s.db.NamedExecContext(ctx, q, fromBusUser(usr)); err != nil {
		return fmt.Errorf("namedExecContext: %w", err)
	}
	return nil
}

func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (usrBus.User, error) {

	data := map[string]any{
		"id": id.String(),
	}

	const q = `SELECT * FROM users WHERE id = :id`

	ctx, span := s.tracer.Start(ctx, "user.store.queryByID")
	defer span.End()

	var usr user

	rows, err := s.db.NamedQueryContext(ctx, q, data)
	if err != nil {
		return usrBus.User{}, fmt.Errorf("namedQueryContext: %w", err)
	}

	defer rows.Close()

	// returns "true" if it was able to move to first row, false if not and means there is no rows
	if !rows.Next() {
		return usrBus.User{}, usrBus.ErrUserNotFound
	}
	//if "rows.Next()" successfully moved to the first row, we can scan it
	if err := rows.StructScan(&usr); err != nil {
		return usrBus.User{}, fmt.Errorf("structScan: %w", err)
	}

	return toUserBus(usr), nil
}

func (s *Store) QueryByEmail(ctx context.Context, email mail.Address) (usrBus.User, error) {
	data := struct {
		Email string `db:"email"`
	}{
		Email: email.Address,
	}

	const q = `SELECT * FROM users WHERE email = :email;`

	ctx, span := s.tracer.Start(ctx, "user.store.queryByEmail")
	defer span.End()

	rows, err := s.db.NamedQueryContext(ctx, q, data)
	if err != nil {
		return usrBus.User{}, fmt.Errorf("namedQueryContext: %w", err)
	}

	defer rows.Close()

	//move cursor to the first rows
	if !rows.Next() {
		return usrBus.User{}, usrBus.ErrUserNotFound
	}

	var usr user
	if err := rows.StructScan(&usr); err != nil {
		return usrBus.User{}, fmt.Errorf("structScan: %w", err)
	}

	return toUserBus(usr), nil
}

func (s *Store) Query(ctx context.Context, filters usrBus.QueryFilter, orderBy usrBus.Field, page page.Page) ([]usrBus.User, error) {
	data := map[string]any{
		"offset":        (page.Number - 1) * page.Rows,
		"rows_per_page": page.Rows,
	}

	const q = "SELECT * FROM users "
	buf := bytes.NewBufferString(q)

	//applying filters
	applyFilters(filters, data, buf)

	//applying order by
	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, fmt.Errorf("orderByClause: %w", err)
	}

	buf.WriteString(orderClause)

	//applying pagination
	buf.WriteString(" OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY;")

	ctx, span := s.tracer.Start(ctx, "user.store.query")
	defer span.End()

	var usrs []user
	rows, err := s.db.NamedQueryContext(ctx, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedQueryContext: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var usr user
		if err := rows.StructScan(&usr); err != nil {
			return nil, fmt.Errorf("structScan: %w", err)
		}
		usrs = append(usrs, usr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("preparing next row to scan: %w", err)
	}

	busUsers := make([]usrBus.User, len(usrs))
	for i, usr := range usrs {
		busUsers[i] = toUserBus(usr)
	}

	return busUsers, nil
}

func (s *Store) Count(ctx context.Context, filters usrBus.QueryFilter) (int, error) {
	const q = `SELECT COUNT(1) FROM users`
	ctx, span := s.tracer.Start(ctx, "user.store.count")
	defer span.End()

	buf := bytes.NewBufferString(q)

	data := map[string]any{}

	applyFilters(filters, data, buf)

	var count struct {
		Count int `db:"count"`
	}

	rows, err := s.db.NamedQueryContext(ctx, buf.String(), data)
	if err != nil {
		return 0, fmt.Errorf("namedQueryContext: %w", err)
	}

	defer rows.Close()

	if !rows.Next() {
		return 0, fmt.Errorf("moving cursor to next row: %w", err)
	}

	if err := rows.StructScan(&count); err != nil {
		return 0, fmt.Errorf("structScan: %w", err)
	}

	return count.Count, nil
}
