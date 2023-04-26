package analytics

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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
	ParseAndStore(context.Context, uuid.UUID, string) error
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

func (r *analyticsRepository) ParseAndStore(ctx context.Context, id uuid.UUID, result string) error {
	log.Ctx(ctx).Trace().Msgf("execution %s parsing and storing results: %s", id.String(), result)

	f := strings.NewReader(result)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		dec := json.NewDecoder(strings.NewReader(line))
		for {
			var resultMap map[string]string
			if err := dec.Decode(&resultMap); err != nil {
				if err == io.EOF {
					break
				}
				break
			}
			for k, v := range resultMap {
				log.Ctx(ctx).Trace().Msgf("execution %s storing result metadata: %s=%s", id.String(), k, v)
				err := r.database.CreateResultMetadata(ctx, db.CreateResultMetadataParams{
					QueueItemID: id,
					Type:        k,
					Value:       v,
				})
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
