package userdb

import (
	"bytes"
	"fmt"
	"strings"

	usrbus "github.com/hamidoujand/jumble/internal/domains/user/bus"
)

func applyFilters(filters usrbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	var whereClause []string

	if filters.Name != nil {
		//first add to sqlx data map
		data["name"] = fmt.Sprintf("%%%s%%", *filters.Name)
		//then add to the where clause
		whereClause = append(whereClause, "name LIKE :name")
	}

	if filters.Department != nil {
		data["department"] = filters.Department
		whereClause = append(whereClause, "department = :department")
	}

	if filters.Role != nil {
		data["roles"] = filters.Role
		whereClause = append(whereClause, "roles && :roles")
	}

	if filters.StartCreatedAt != nil {
		data["start_created_at"] = filters.StartCreatedAt.UTC()
		whereClause = append(whereClause, "created_at >= :start_created_at")
	}

	if filters.EndCreatedAt != nil {
		data["end_created_at"] = filters.EndCreatedAt.UTC()
		whereClause = append(whereClause, "created_at <= :end_created_at")
	}

	//join all of them with " AND "
	if len(whereClause) > 0 {
		buf.WriteString(" WHERE ")
		q := strings.Join(whereClause, " AND ")
		buf.WriteString(q)
	}
}
