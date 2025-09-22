package userdb

import (
	"fmt"

	usrBus "github.com/hamidoujand/jumble/internal/domains/user/bus"
)

// translates field names from Bus layer to Store valid fields.
var orderByFieldNames = map[string]string{
	usrBus.OrderByName:      "name",
	usrBus.OrderByEmail:     "email",
	usrBus.OrderByCreatedAt: "created_at",
	usrBus.OrderByUpdatedAt: "updated_at",
}

func orderByClause(field usrBus.Field) (string, error) {
	by, ok := orderByFieldNames[field.Name]
	if !ok {
		return "", fmt.Errorf("%q is not a valid field to order by", field.Name)
	}

	return " ORDER BY " + by + " " + field.Dir, nil
}
