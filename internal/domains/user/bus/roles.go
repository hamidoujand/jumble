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

// RoleSlice - custom type for PostgreSQL TEXT[] handling
type RoleSlice []Role

// Scan implements sql.Scanner - reads from database into []Role
func (rs *RoleSlice) Scan(val any) error {
	if val == nil {
		//return an empty slice
		*rs = RoleSlice{}
		return nil
	}

	switch v := val.(type) {
	case []byte:
		return rs.parsePostgresArray(string(v))
	case string:
		return rs.parsePostgresArray(v)
	default:
		return fmt.Errorf("unsupported type for role slice: %T", v)
	}

}

// Value implements driver.Valuer - writes []Role to database
func (rs RoleSlice) Value() (driver.Value, error) {
	if len(rs) == 0 {
		return "{}", nil
	}

	// Format: {"admin","user"}
	qouted := make([]string, len(rs))

	for i, role := range rs {
		escaped := strings.ReplaceAll(role.String(), `"`, `""`)
		qouted[i] = `"` + escaped + `"`
	}
	return "{" + strings.Join(qouted, ",") + "}", nil
}

func (rs *RoleSlice) parsePostgresArray(arr string) error {
	// remove the outer braces {}
	s := strings.Trim(arr, "{}")

	//if the arr is empty
	if s == "" {
		*rs = RoleSlice{}
		return nil
	}

	//parser for comma-separated values within braces
	var elements []string
	var current strings.Builder
	inQoutes := false
	escapeNext := false

	for _, ch := range s {
		switch {
		case escapeNext:
			current.WriteRune(ch)
			escapeNext = false
		case ch == '\\':
			escapeNext = true
		case ch == '"':
			inQoutes = !inQoutes
		case ch == ',' && !inQoutes:
			elements = append(elements, current.String())
			current.Reset()
		default:
			current.WriteRune(ch)
		}
	}

	//do not forget about last element
	if current.Len() > 0 {
		elements = append(elements, current.String())
	}

	//parse string elements to Role
	roles := make([]Role, len(elements))
	for i, elem := range elements {
		//remove any qoutes around them
		elem = strings.Trim(elem, `"`)

		r, err := parseRole(elem)
		if err != nil {
			return fmt.Errorf("parseRole from DB: %w", err)
		}

		roles[i] = r
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
