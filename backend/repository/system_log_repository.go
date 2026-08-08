package repository

import (
	"context"

	"music-discovery/backend/entity"
)

// ISystemLogRepository defines the persistence operations required by the
// system log use case. It is kept in the repository package consistently
// with the other repository ports in this project.
type ISystemLogRepository interface {
	CreateSystemLog(context.Context, *entity.SystemLog) error
}
