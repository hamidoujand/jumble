package bus

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

var (
	RoleAdmin = newRole("admin")
	RoleUser  = newRole("user")
)

// Role represents a role in our system, since requires some validation, created a new custom type for it.
type Role struct {
	value string
}

var validRoles = make(map[string]Role)

func newRole(val string) Role {
	r := Role{value: val}

	validRoles[val] = r
	return r
}

func (r Role) String() string {
	return r.value
}

func (r Role) MarshalText() ([]byte, error) {
	return []byte(r.value), nil
}

//==============================================================================

// RoleSlice is a custom type that implements sql.Scanner and driver.Valuer
// to handle PostgreSQL TEXT[] arrays for our Role type
type RoleSlice []Role

// Scan implements sql.Scanner - converts database TEXT[] to Go []Role
func (rs *RoleSlice) Scan(val any) error {
	// Handle NULL values from database
	if val == nil {
		//return an empty Role slice instead of nil.
		*rs = RoleSlice{}
		return nil
	}

	// Handle different types the database might return
	switch v := val.(type) {
	case []byte:
		return rs.parsePostgresArray(string(v))
	case string:
		return rs.parsePostgresArray(v)
	default:
		return fmt.Errorf("unsupported type for role slice: %T", v)
	}

}

// Value implements driver.Valuer - converts Go []Role to PostgreSQL TEXT[]
func (rs RoleSlice) Value() (driver.Value, error) {
	// Handle empty slice
	if len(rs) == 0 {
		return "{}", nil //PostgreSQL empty array literal
	}

	// Format each role with proper quoting for PostgreSQL array
	//since Roles are defined by App not user input then it is a list of comma-separated values
	qouted := make([]string, len(rs))

	for i, role := range rs {
		qouted[i] = role.String()
	}
	return "{" + strings.Join(qouted, ",") + "}", nil
}

// Example input from database: "{admin,user}" or "{\"admin\",\"user\"}"
func (rs *RoleSlice) parsePostgresArray(arr string) error {
	// Remove outer braces: "{admin,user}" -> "admin,user"
	s := strings.Trim(arr, "{}")

	//if the arr is empty
	if s == "" {
		*rs = RoleSlice{}
		return nil
	}

	// Simple split on comma - no need for complex parsing since the Roles are defined by the App not user input.
	elements := strings.Split(s, ",")
	roles := make([]Role, len(elements))
	for i, elem := range elements {
		// Remove any quotes (in case PostgreSQL added them)
		elem = strings.Trim(elem, `"`)

		// Validate the role
		role, err := parseRole(elem)
		if err != nil {
			return fmt.Errorf("invalid role in array: %w", err)
		}
		roles[i] = role
	}

	*rs = roles
	return nil
}

// ------------------------------------------------------------------------------
func parseRole(val string) (Role, error) {
	r, ok := validRoles[val]
	if !ok {
		return Role{}, fmt.Errorf("invalid role: %s", val)
	}

	return r, nil
}

func RolesToString(roles []Role) []string {
	res := make([]string, len(roles))

	for i, r := range roles {
		res[i] = r.value
	}

	return res
}

func ParseManyRoles(rr []string) ([]Role, error) {
	res := make([]Role, len(rr))

	for i, r := range rr {
		role, err := parseRole(r)
		if err != nil {
			return nil, err
		}
		res[i] = role
	}

	return res, nil
}
