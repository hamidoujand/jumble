package handler

import (
	"time"

	"github.com/hamidoujand/jumble/internal/domains/user/bus"
)

type Filters struct {
	Name           *string  `form:"name" binding:"omitempty,min=4,max=120"`
	Department     *string  `form:"department" binding:"omitempty,oneof=sales marketing"`
	Roles          []string `form:"roles" binding:"omitempty,dive,oneof=user admin"`
	StartCreatedAt *string  `form:"startCreatedAt" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"` //RFC3339
	EndCreatedAt   *string  `form:"endCreatedAt" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`   //RFC3339
}

func (f Filters) ToBusQueryFilter() (bus.QueryFilter, error) {
	var busQueryFilters bus.QueryFilter
	if len(f.Roles) != 0 {
		parsedRoles, err := bus.ParseManyRoles(f.Roles)
		if err != nil {
			return bus.QueryFilter{}, err
		}

		busQueryFilters.Roles = parsedRoles
	}

	if f.Department != nil {
		busQueryFilters.Department = f.Department
	}

	if f.Name != nil {
		busQueryFilters.Name = f.Name
	}

	if f.StartCreatedAt != nil {
		start, err := time.Parse(time.RFC3339, *f.StartCreatedAt)
		if err != nil {
			return bus.QueryFilter{}, err
		}
		busQueryFilters.StartCreatedAt = &start
	}

	if f.EndCreatedAt != nil {
		end, err := time.Parse(time.RFC3339, *f.EndCreatedAt)
		if err != nil {
			return bus.QueryFilter{}, err
		}
		busQueryFilters.EndCreatedAt = &end
	}

	return busQueryFilters, nil
}
