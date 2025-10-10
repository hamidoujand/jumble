package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/jumble/internal/auth"
	"github.com/hamidoujand/jumble/internal/dbtest"
	"github.com/hamidoujand/jumble/internal/domains/user/bus"
	"github.com/hamidoujand/jumble/internal/domains/user/store/userdb"
	"github.com/hamidoujand/jumble/internal/errs"
	"github.com/hamidoujand/jumble/pkg/docker"
	"github.com/hamidoujand/jumble/pkg/mux"
	"go.opentelemetry.io/otel"
)

var container docker.Container

func TestMain(m *testing.M) {
	// before all
	var err error
	container, err = dbtest.CreateDBContainer()
	if err != nil {
		os.Exit(1)
	}

	// tests
	code := m.Run()

	// after all
	docker.StopContainer(container.Name)

	os.Exit(code)
}

func Test_CreateUser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		newUser    newUser
		expectErr  bool
		isModelErr bool
		statusCode int
	}{
		{
			name: "create_user_201",
			newUser: newUser{
				Name:            "John Doe",
				Email:           "john@doe.com",
				Roles:           []string{"user"},
				Department:      "sales",
				Password:        "test1234",
				PasswordConfirm: "test1234",
			},
			expectErr:  false,
			isModelErr: false,
			statusCode: http.StatusCreated,
		},
		{
			name: "create_user_400",
			newUser: newUser{
				Name:            "Jo",
				Email:           "john.com",
				Roles:           []string{"super Admin"},
				Department:      "loading",
				Password:        "test",
				PasswordConfirm: "test1",
			},
			expectErr:  true,
			isModelErr: true,
			statusCode: http.StatusBadRequest,
		},
		{
			name: "create_user_duplicated_email",
			newUser: newUser{
				Name:            "John Doe",
				Email:           "john@doe.com",
				Roles:           []string{"user"},
				Department:      "sales",
				Password:        "test1234",
				PasswordConfirm: "test1234",
			},
			expectErr:  true,
			isModelErr: false,
			statusCode: http.StatusBadRequest,
		},
	}

	setups := setupPerTest(t)

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {

			var buf bytes.Buffer
			err := json.NewEncoder(&buf).Encode(ts.newUser)
			if err != nil {
				t.Fatalf("failed to encode: %s", err)
			}

			r := httptest.NewRequest(http.MethodPost, "/v1/users", &buf)
			w := httptest.NewRecorder()

			ctx := context.Background()
			ctx = mux.SetReqMetadata(ctx, &mux.RequestMeta{})
			err = setups.h.CreateUser(ctx, w, r)

			if !ts.expectErr {
				//success
				if err != nil {
					t.Fatalf("failed to create user: %s", err)
				}
				if w.Result().StatusCode != ts.statusCode {
					t.Errorf("status=%d, got=%d", ts.statusCode, w.Result().StatusCode)
				}

				var createdUser user
				if err := json.NewDecoder(w.Body).Decode(&createdUser); err != nil {
					t.Fatalf("failed to decode user: %s", err)
				}

				if createdUser.Token == "" {
					t.Error("expected token to be set")
				}
			} else {
				//expect to fail
				if err == nil {
					t.Fatalf("expected to fail: %s", ts.name)
				}

				var apiErr *errs.Error
				if !errors.As(err, &apiErr) {
					t.Fatalf("errorType=%T, got=%T", &errs.Error{}, err)
				}

				if apiErr.Code != ts.statusCode {
					t.Errorf("status=%d, got=%d", ts.statusCode, apiErr.Code)
				}

				if ts.isModelErr {
					expectedFields := []string{"name", "email", "roles", "password", "department", "passwordConfirm"}
					for field := range apiErr.Fields {
						if !slices.Contains(expectedFields, field) {
							t.Errorf("expected field[%s] to fail validation", field)
						}
					}
				}
			}
		})
	}
}

func Test_UpdateUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		newUser    newUser
		updates    updateUser
		expectErr  bool
		isModelErr bool
		statusCode int
	}{
		{
			name: "update_user_200",
			newUser: newUser{
				Name:            "John Doe",
				Email:           "john@doe.com",
				Roles:           []string{"user"},
				Department:      "sales",
				Password:        "test1234",
				PasswordConfirm: "test1234",
			},
			updates: updateUser{
				Name:            newPointer("Jane Doe"),
				Email:           newPointer("jane@doe.com"),
				Deaprtment:      newPointer("marketing"),
				Password:        newPointer("1234test"),
				PasswordConfirm: newPointer("1234test"),
				Enabled:         newPointer(false),
			},
			expectErr:  false,
			isModelErr: false,
			statusCode: http.StatusOK,
		},
		{
			name: "update_user_400",
			newUser: newUser{
				Name:            "John Doe",
				Email:           "john@doe.com",
				Roles:           []string{"user"},
				Department:      "sales",
				Password:        "test1234",
				PasswordConfirm: "test1234",
			},
			updates: updateUser{
				Name:            newPointer("Ja"),
				Email:           newPointer("janedoe.com"),
				Deaprtment:      newPointer("mark"),
				Password:        newPointer("test"),
				PasswordConfirm: newPointer("14tesst"),
				Enabled:         newPointer(false),
			},
			expectErr:  true,
			isModelErr: true,
			statusCode: http.StatusBadRequest,
		},
	}

	setup := setupPerTest(t)

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			//insert the user into db
			busUser, err := toBusNewUser(ts.newUser)
			if err != nil {
				t.Fatalf("failed toBusNewUser: %s", err)
			}

			created, err := setup.userBus.Create(context.Background(), busUser)
			if err != nil {
				t.Fatalf("failed to create new user: %s", err)
			}

			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(ts.updates); err != nil {
				t.Fatalf("failed to encode updates to json: %s", err)
			}

			p := fmt.Sprintf("/v1/users/%s", created.ID)
			r := httptest.NewRequest(http.MethodPut, p, &buf)
			//set the path param to request
			r.SetPathValue("id", created.ID.String())
			w := httptest.NewRecorder()
			//set request metadata
			ctx := context.Background()
			ctx = mux.SetReqMetadata(ctx, &mux.RequestMeta{})

			//set the user making the request inside ctx
			ctx = auth.SetUser(ctx, created)

			err = setup.h.UpdateUser(ctx, w, r)
			if !ts.expectErr {
				// expected to succeed
				if err != nil {
					t.Fatalf("expected to update the user: %s", err)
				}

				gotStatus := w.Result().StatusCode
				if gotStatus != ts.statusCode {
					t.Errorf("status=%d, got=%d", ts.statusCode, gotStatus)
				}

				var updatedUser user
				if err := json.NewDecoder(w.Body).Decode(&updatedUser); err != nil {
					t.Fatalf("failed to decode updated user from response: %s", err)
				}

				if updatedUser.Name != *ts.updates.Name {
					t.Errorf("name=%s, got=%s", *ts.updates.Name, updatedUser.Name)
				}

				if updatedUser.Email != *ts.updates.Email {
					t.Errorf("email=%s, got=%s", *ts.updates.Email, updatedUser.Email)
				}

				if updatedUser.Department != *ts.updates.Deaprtment {
					t.Errorf("department=%s, got=%s", *ts.updates.Deaprtment, updatedUser.Department)
				}

				if updatedUser.Enabled != *ts.updates.Enabled {
					t.Errorf("enabled=%t, got=%t", *ts.updates.Enabled, updatedUser.Enabled)
				}
			} else {
				//expect to fail
				if err == nil {
					t.Fatalf("expected test to fail: %s", ts.name)
				}
				var apiErr *errs.Error
				if !errors.As(err, &apiErr) {
					t.Fatalf("errorType=%T, got=%T", &errs.Error{}, err)
				}

				if apiErr.Code != ts.statusCode {
					t.Errorf("status=%d, got=%d", ts.statusCode, apiErr.Code)
				}

				if ts.isModelErr {
					expectedFields := []string{"name", "email", "password", "department", "passwordConfirm"}
					for field := range apiErr.Fields {
						if !slices.Contains(expectedFields, field) {
							t.Errorf("expected field[%s] to fail validation", field)
						}
					}
				}
			}

			//clean the db for next test
			if err := setup.userBus.Delete(context.Background(), created); err != nil {
				t.Fatalf("expected to clean the users table: %s", err)
			}
		})
	}
}

func Test_Query(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		statusCode         int
		query              string
		expectedNumRecords int
		isOrderBy          bool
	}{
		{
			name:               "fetch_alex",
			query:              "/v1/users?name=Alex&page=1&rows=1",
			expectedNumRecords: 1,
			statusCode:         http.StatusOK,
		},
		{
			name:               "fetch_all_admins",
			query:              "/v1/users?roles=admin",
			expectedNumRecords: 2,
			statusCode:         http.StatusOK,
		},
		{
			name:               "fetch_all_admins_and_users",
			query:              "/v1/users?roles=admin&roles=user",
			expectedNumRecords: 4,
			statusCode:         http.StatusOK,
		},
		{
			name:               "fetch_all_admins_and_users_page_1_rows_1",
			query:              "/v1/users?roles=admin&roles=user&page=1&rows=1",
			expectedNumRecords: 1,
			statusCode:         http.StatusOK,
		},
		{
			name:               "fetch_all_admins_and_users_order_by_name_DESC",
			query:              "/v1/users?roles=admin&roles=user&page=1&rows=1&order_by=name,desc",
			expectedNumRecords: 1,
			statusCode:         http.StatusOK,
			isOrderBy:          true,
		},
	}

	setup := setupPerTest(t)

	//seed the db
	users := []newUser{
		{
			Name:            "Alex Doe",
			Email:           "alex@doe.com",
			Roles:           []string{"user", "admin"},
			Department:      "sales",
			Password:        "test1234",
			PasswordConfirm: "test1234",
		},
		{
			Name:            "Bob Doe",
			Email:           "bob@doe.com",
			Roles:           []string{"admin"},
			Department:      "sales",
			Password:        "test1234",
			PasswordConfirm: "test1234",
		},
		{
			Name:            "Will Doe",
			Email:           "will@doe.com",
			Roles:           []string{"user"},
			Department:      "marketing",
			Password:        "test1234",
			PasswordConfirm: "test1234",
		},
		{
			Name:            "Zack Doe",
			Email:           "zack@doe.com",
			Roles:           []string{"user"},
			Department:      "marketing",
			Password:        "test1234",
			PasswordConfirm: "test1234",
		},
	}

	for _, usr := range users {
		busUser, err := toBusNewUser(usr)
		if err != nil {
			t.Fatalf("toBusUser: %s", err)
		}

		_, err = setup.userBus.Create(context.Background(), busUser)
		if err != nil {
			t.Fatalf("create: %s", err)
		}
	}

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPut, ts.query, nil)
			w := httptest.NewRecorder()
			//set request metadata
			ctx := context.Background()
			ctx = mux.SetReqMetadata(ctx, &mux.RequestMeta{})

			err := setup.h.Query(ctx, w, r)
			if err != nil {
				t.Fatalf("query: %s", err)
			}

			status := w.Result().StatusCode
			if status != ts.statusCode {
				t.Errorf("status=%d, got=%d", ts.statusCode, status)
			}

			var queryResult QueryResult
			if err := json.NewDecoder(w.Body).Decode(&queryResult); err != nil {
				t.Fatalf("decode: %s", err)
			}

			if len(queryResult.Users) != ts.expectedNumRecords {
				t.Errorf("total=%d, got=%d", ts.expectedNumRecords, queryResult.Total)
			}

			if ts.isOrderBy {
				//zack
				zack := queryResult.Users[0]
				if zack.Name != "Zack Doe" {
					t.Errorf("name=%s, got=%s", "Zack Doe", zack.Name)
				}
			}
		})
	}
}

// =============================================================================

type setup struct {
	h       handler
	userBus *bus.Bus
}

func setupPerTest(t *testing.T) setup {
	db := dbtest.New(t, container, "create_user_api")
	tracer := otel.Tracer("user_tests")
	usrStore := userdb.NewStore(db, tracer)
	usrBus := bus.New(usrStore)

	ks := newKeyStore(t)
	issuer := "jumple_tests"
	a := auth.New(ks, usrBus, issuer)
	kid := uuid.NewString()

	h := handler{
		userBus:     usrBus,
		a:           a,
		kid:         kid,
		issuer:      issuer,
		tokenMaxAge: time.Minute,
	}

	return setup{
		h:       h,
		userBus: usrBus,
	}
}

// ==============================================================================
func newPointer[T any](val T) *T {
	return &val
}

// ==============================================================================
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
