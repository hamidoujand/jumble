package bus

import (
	"time"
)

type QueryFilter struct {
	Name           *string
	Department     *string
	Role           []Role
	StartCreatedAt *time.Time
	EndCreatedAt   *time.Time
}
