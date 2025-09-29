package handler

import (
	"net/http"
	"time"

	"github.com/hamidoujand/jumble/internal/domains/user/bus"
	"github.com/hamidoujand/jumble/internal/errs"
)

type Filters struct {
	Name           *string  `validate:"omitempty,min=4,max=120"`
	Department     *string  `validate:"omitempty,oneof=sales marketing"`
	Roles          []string `validate:"omitempty,dive,oneof=user admin"`
	StartCreatedAt *string  `validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"` //RFC3339
	EndCreatedAt   *string  `validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"` //RFC3339
}

func parseFilters(r *http.Request) (bus.QueryFilter, error) {
	q := r.URL.Query()
	var filters Filters

	//setup for validation
	name := q.Get("name")
	department := q.Get("department")
	roles := q["roles"] //slice or string
	startCreatedAt := q.Get("start_created_at")
	endCreatedAt := q.Get("end_created_at")

	if name != "" {
		filters.Name = &name
	}

	if department != "" {
		filters.Department = &department
	}

	if len(roles) != 0 {
		filters.Roles = roles
	}
	if startCreatedAt != "" {
		filters.StartCreatedAt = &startCreatedAt
	}

	if endCreatedAt != "" {
		filters.EndCreatedAt = &endCreatedAt
	}

	//validate it
	fileds := errs.Check(filters)
	if len(fileds) != 0 {
		return bus.QueryFilter{}, errs.NewValidationErr(http.StatusBadRequest, fileds)
	}

	//after validation assign to bus query filters
	var busQueryFilters bus.QueryFilter
	if len(filters.Roles) != 0 {
		parsedRoles, err := bus.ParseManyRoles(roles)
		if err != nil {
			return bus.QueryFilter{}, errs.NewValidationErr(http.StatusBadRequest, map[string]string{"roles": err.Error()})
		}

		busQueryFilters.Roles = parsedRoles
	}

	if filters.Department != nil {
		busQueryFilters.Department = filters.Department
	}

	if filters.Name != nil {
		busQueryFilters.Name = filters.Name
	}

	if filters.StartCreatedAt != nil {
		start, err := time.Parse(time.RFC3339, *filters.StartCreatedAt)
		if err != nil {
			return bus.QueryFilter{}, errs.NewValidationErr(http.StatusBadRequest, map[string]string{"start_created_at": err.Error()})
		}
		busQueryFilters.StartCreatedAt = &start
	}

	if filters.EndCreatedAt != nil {
		end, err := time.Parse(time.RFC3339, *filters.EndCreatedAt)
		if err != nil {
			return bus.QueryFilter{}, errs.NewValidationErr(http.StatusBadRequest, map[string]string{"end_created_at": err.Error()})
		}
		busQueryFilters.EndCreatedAt = &end
	}

	return busQueryFilters, nil
}
