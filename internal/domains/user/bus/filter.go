package bus

import (
	"time"
)

type QueryFilter struct {
	Name           *string
	Department     *string
	Roles          []Role
	StartCreatedAt *time.Time
	EndCreatedAt   *time.Time
}
