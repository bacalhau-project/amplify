package analytics

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/pkg/errors"
)

var (
	ErrAnalyticsErr    = fmt.Errorf("analytics error")
	ErrInvalidPageSize = errors.Wrap(ErrAnalyticsErr, "invalid page size")
	ErrInvalidKey      = errors.Wrap(ErrAnalyticsErr, "invalid key")
)

type analyticsRepository struct {
	database db.Analytics
}
type AnalyticsRepository interface {
	QueryTopResultsByKey(ctx context.Context, params QueryTopResultsByKeyParams) (map[string]interface{}, error)
}

func NewAnalyticsRepository(d db.Analytics) AnalyticsRepository {
	return &analyticsRepository{
		database: d,
	}
}

type QueryTopResultsByKeyParams struct {
	Key      string
	PageSize int
}

func (r *analyticsRepository) QueryTopResultsByKey(ctx context.Context, params QueryTopResultsByKeyParams) (map[string]interface{}, error) {
	if params.PageSize <= 0 {
		return nil, ErrInvalidPageSize
	}
	if params.Key == "" {
		return nil, ErrInvalidKey
	}
	rows, err := r.database.QueryTopResultsByKey(ctx, db.QueryTopResultsByKeyParams{
		Key:      params.Key,
		PageSize: int32(params.PageSize),
	})
	if err != nil {
		return nil, err
	}
	results := make(map[string]interface{}, params.PageSize)
	for _, row := range rows {
		results[row.Value] = row.Count
	}
	return results, nil
}
