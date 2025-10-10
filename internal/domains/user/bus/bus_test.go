package bus_test

import (
	"context"
	"errors"
	"log"
	"net/mail"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hamidoujand/jumble/internal/dbtest"
	"github.com/hamidoujand/jumble/internal/domains/user/bus"
	"github.com/hamidoujand/jumble/internal/domains/user/store/userdb"
	"github.com/hamidoujand/jumble/internal/page"
	"github.com/hamidoujand/jumble/pkg/docker"
	"github.com/hamidoujand/jumble/pkg/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
)

var container docker.Container
var tracer trace.Tracer

func TestMain(m *testing.M) {
	// before all
	var err error
	container, err = dbtest.CreateDBContainer()
	if err != nil {
		log.Fatalf("createDBContainer: %s", err)
	}

	defer docker.StopContainer(container.Name)
	cfg := telemetry.Config{
		ServiceName: "user_bus_test",
		Host:        "",
		Build:       "v0.0.1",
	}

	cleanup, err := telemetry.SetupOTelSDK(cfg)
	if err != nil {
		log.Fatalf("setupOTelSDK: %s", err)
	}

	tracer = otel.Tracer("user_bus_tests")

	defer cleanup(context.Background())

	// tests
	os.Exit(m.Run())

}

func Test_CreateUser(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, container, "create_user")
	store := userdb.NewStore(db, tracer)

	b := bus.New(store)

	nu := bus.NewUser{
		Name: "John Doe",
		Email: mail.Address{
			Name:    "John Doe",
			Address: "john@gmail.com",
		},
		Roles:      []bus.Role{bus.RoleUser},
		Department: "Sales",
		Password:   "test1234",
	}

	usr, err := b.Create(context.Background(), nu)
	if err != nil {
		t.Fatalf("failed to create a user: %s", err)
	}

	//check password
	if err := bcrypt.CompareHashAndPassword(usr.PasswordHash, []byte(nu.Password)); err != nil {
		t.Fatalf("passwords did not match")
	}
}

func Test_GetByID(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, container, "get_user_by_id")
	store := userdb.NewStore(db, tracer)

	b := bus.New(store)

	nu := bus.NewUser{
		Name: "John Doe",
		Email: mail.Address{
			Name:    "John Doe",
			Address: "john@gmail.com",
		},
		Roles:      []bus.Role{bus.RoleUser},
		Department: "Sales",
		Password:   "test1234",
	}

	usr, err := b.Create(context.Background(), nu)
	if err != nil {
		t.Fatalf("failed to create a user: %s", err)
	}

	fetched, err := b.QueryByID(context.Background(), usr.ID)
	if err != nil {
		t.Fatalf("failed to query by id: %s", err)
	}

	// Allow unexported fields for the Role type
	opts := []cmp.Option{
		cmp.AllowUnexported(bus.Role{}),
	}
	if diff := cmp.Diff(fetched, usr, opts...); diff != "" {
		t.Errorf("mismatch (-got +want):\n%s", diff)
	}
}

func Test_UpdateUser(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, container, "update_user")
	store := userdb.NewStore(db, tracer)

	b := bus.New(store)

	nu := bus.NewUser{
		Name: "John Doe",
		Email: mail.Address{
			Name:    "John Doe",
			Address: "john@gmail.com",
		},
		Roles:      []bus.Role{bus.RoleUser},
		Department: "Sales",
		Password:   "test1234",
	}

	usr, err := b.Create(context.Background(), nu)
	if err != nil {
		t.Fatalf("failed to create a user: %s", err)
	}

	name := "Jane Doe"
	email := mail.Address{Name: "Jane Doe", Address: "jane@gmail.com"}
	pass := "test54321"
	role := []bus.Role{bus.RoleAdmin}
	department := "shipping"
	enabled := false
	uu := bus.UpdateUser{
		Name:       &name,
		Email:      &email,
		Roles:      role,
		Department: &department,
		Password:   &pass,
		Enabled:    &enabled,
	}

	updated, err := b.Update(context.Background(), usr, uu)
	if err != nil {
		t.Fatalf("update failed: %s", err)
	}

	if updated.Name != name {
		t.Errorf("name=%s, got=%s", name, updated.Name)
	}

	if updated.Department != department {
		t.Errorf("department=%s, got=%s", department, updated.Department)
	}

	if updated.Roles[0] != role[0] {
		t.Errorf("role=%s, got=%s", role[0], updated.Roles[0])
	}

	if updated.Email.Address != email.Address {
		t.Errorf("email=%s, got=%s", email.Address, updated.Email.Address)
	}

	if err := bcrypt.CompareHashAndPassword(updated.PasswordHash, []byte(pass)); err != nil {
		t.Errorf("password does not match: %s", err)
	}

	if updated.UpdatedAt.Equal(usr.CreatedAt) {
		t.Errorf("updated at should not equal to created at")
	}
}

func Test_DeleteUser(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, container, "delete_user")
	store := userdb.NewStore(db, tracer)

	b := bus.New(store)

	nu := bus.NewUser{
		Name: "John Doe",
		Email: mail.Address{
			Name:    "John Doe",
			Address: "john@gmail.com",
		},
		Roles:      []bus.Role{bus.RoleUser},
		Department: "Sales",
		Password:   "test1234",
	}

	usr, err := b.Create(context.Background(), nu)
	if err != nil {
		t.Fatalf("failed to create a user: %s", err)
	}

	if err := b.Delete(context.Background(), usr); err != nil {
		t.Fatalf("failed to delete user: %s", err)
	}

	_, err = b.QueryByID(context.Background(), usr.ID)
	if err == nil {
		t.Fatal("expected user to be deleted")
	}

	if !errors.Is(err, bus.ErrUserNotFound) {
		t.Errorf("err=%s, got=%s", bus.ErrUserNotFound, err)
	}
}

func Test_QueryByEmail(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, container, "get_user_by_email")
	store := userdb.NewStore(db, tracer)

	b := bus.New(store)

	nu := bus.NewUser{
		Name: "John Doe",
		Email: mail.Address{
			Name:    "John Doe",
			Address: "john@gmail.com",
		},
		Roles:      []bus.Role{bus.RoleUser},
		Department: "Sales",
		Password:   "test1234",
	}

	usr, err := b.Create(context.Background(), nu)
	if err != nil {
		t.Fatalf("failed to create a user: %s", err)
	}

	fetched, err := b.QueryByEmail(context.Background(), usr.Email)
	if err != nil {
		t.Fatalf("failed to query by email: %s", err)
	}

	//compare users
	diff := cmp.Diff(usr, fetched, cmp.AllowUnexported(bus.Role{}))
	if diff != "" {
		t.Logf("Diff: %s\n", diff)
	}
}

func Test_Query(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, container, "query_users")

	store := userdb.NewStore(db, tracer)
	b := bus.New(store)
	querySetup(t, b)

	//query by name and department
	name := "John Doe"
	department := "Sales"

	f := bus.QueryFilter{
		Name:       &name,
		Department: &department,
	}
	o, err := bus.ParseOrderBy("name,asc")
	if err != nil {
		t.Fatalf("expected to parse order by clause: %s", err)
	}

	p, err := page.Parse("1", "1")
	if err != nil {
		t.Fatalf("expected to parse page: %s", err)
	}

	usr, err := b.Query(context.Background(), f, o, p)
	if err != nil {
		t.Fatalf("failed to query: %s", err)
	}

	if len(usr) != 1 {
		t.Fatalf("expected to find at least one user, got=%d", len(usr))
	}

	john := usr[0]

	if john.Name != name {
		t.Errorf("name=%s, got-%s", name, john.Name)
	}

	if john.Department != department {
		t.Errorf("department=%s, got=%s", department, john.Department)
	}

	//get all Admins
	role := []bus.Role{bus.RoleAdmin}

	f = bus.QueryFilter{
		Roles: role,
	}

	o, err = bus.ParseOrderBy("name,desc")
	if err != nil {
		t.Fatalf("parsing order by failed: %s", err)
	}

	p, err = page.Parse("1", "2")
	if err != nil {
		t.Fatalf("parsing page failed: %s", err)
	}

	admins, err := b.Query(t.Context(), f, o, p)
	if err != nil {
		t.Fatalf("query failed: %s", err)
	}

	if len(admins) != 2 {
		t.Fatalf("admins=%d, got=%d", 2, len(admins))
	}

	//first one should be Tom Doe because of "order by name desc"
	mike := admins[0]
	name = "Tom Doe"
	if mike.Name != name {
		t.Errorf("name=%s, got=%s", name, mike.Name)
	}
}

// ==============================================================================
func querySetup(t *testing.T, b *bus.Bus) {
	nus := []bus.NewUser{
		{
			Name: "John Doe",
			Email: mail.Address{
				Name:    "John Doe",
				Address: "john@doe.com",
			},
			Roles:      []bus.Role{bus.RoleUser},
			Department: "Sales",
			Password:   "test123",
		},
		{
			Name: "Jane Doe",
			Email: mail.Address{
				Name:    "Jane Doe",
				Address: "Jane@doe.com",
			},
			Roles:      []bus.Role{bus.RoleUser},
			Department: "Shipping",
			Password:   "test123",
		},
		{
			Name: "Mike Doe",
			Email: mail.Address{
				Name:    "Mike Doe",
				Address: "mike@doe.com",
			},
			Roles:      []bus.Role{bus.RoleAdmin},
			Department: "Sales",
			Password:   "test123",
		},
		{
			Name: "Tom Doe",
			Email: mail.Address{
				Name:    "Tom Doe",
				Address: "tom@doe.com",
			},
			Roles:      []bus.Role{bus.RoleAdmin},
			Department: "Shipping",
			Password:   "test123",
		},
	}

	for _, usr := range nus {
		_, err := b.Create(context.Background(), usr)
		if err != nil {
			t.Fatalf("failed to create user: %s: %s", usr.Name, err)
		}
	}

}
