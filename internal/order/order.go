package order

import (
	"fmt"
	"strings"
)

// directions
const (
	ASC  = "ASC"
	DESC = "DESC"
)

// set of directions
var dirs = map[string]string{
	ASC:  "ASC",
	DESC: "DESC",
}

type Field struct {
	Val       string
	Direction string
}

func NewField(field string, dir string) Field {
	if _, ok := dirs[dir]; !ok {
		return Field{
			Val:       field,
			Direction: ASC, //defaultto asc
		}
	}
	return Field{Val: field, Direction: dir}
}

// Parse constructs a field from query string like "field,direction" example "age,desc"
func Parse(domainFields map[string]string, query string, domainDefaultOrder Field) (Field, error) {
	if query == "" {
		//send the defaults for each domain
		return domainDefaultOrder, nil
	}

	orderParts := strings.Split(query, ",")
	fieldName := strings.TrimSpace(orderParts[0])

	//check for valid field name based on domain fields
	validField, ok := domainFields[fieldName]
	if !ok {
		return Field{}, fmt.Errorf("unknown field: %s", fieldName)
	}

	//check to see if user provided dir as well or not
	switch len(orderParts) {
	case 1:
		//only field name provided
		return NewField(validField, ASC), nil
	case 2:
		dir := orderParts[1]
		validDir, ok := dirs[dir]
		if !ok {
			return Field{}, fmt.Errorf("unknown direction: %s", dir)
		}

		return NewField(validField, validDir), nil
	default:
		return Field{}, fmt.Errorf("unknown order: %s", query)
	}
}
