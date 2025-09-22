package bus

import (
	"fmt"
	"strings"
)

const (
	OrderByName      = "name"
	OrderByEmail     = "email"
	OrderByCreatedAt = "createdAt"
	OrderByUpdatedAt = "updatedAt"
)

const (
	OrderByASC  = "asc"
	OrderByDESC = "desc"
)

var orderBySet = map[string]string{
	OrderByName:      "name",
	OrderByEmail:     "email",
	OrderByCreatedAt: "createdAt",
	OrderByUpdatedAt: "updatedAt",
}

var directionsSet = map[string]string{
	OrderByASC:  "asc",
	OrderByDESC: "desc",
}

type Field struct {
	Name string
	Dir  string
}

func ParseOrderBy(query string) (Field, error) {
	if query == "" {
		//return default
		return Field{Name: OrderByCreatedAt, Dir: OrderByASC}, nil
	}

	orderParts := strings.Split(query, ",")
	fieldName := strings.TrimSpace(orderParts[0])

	//check for valid field name based on domain fields
	validField, ok := orderBySet[fieldName]
	if !ok {
		return Field{}, fmt.Errorf("unknown field: %s", fieldName)
	}

	//check to see if user provided dir as well or not
	switch len(orderParts) {
	case 1:
		//only field name provided
		return Field{Name: validField, Dir: OrderByASC}, nil
	case 2:
		dir := orderParts[1]
		validDir, ok := directionsSet[dir]
		if !ok {
			return Field{}, fmt.Errorf("unknown direction: %s", dir)
		}

		return Field{Name: validField, Dir: validDir}, nil
	default:
		return Field{}, fmt.Errorf("unknown order: %s", query)
	}
}
