package analytics

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	ErrAnalyticsErr     = fmt.Errorf("analytics error")
	ErrInvalidPageSize  = errors.Wrap(ErrAnalyticsErr, "invalid page size")
	ErrInvalidKey       = errors.Wrap(ErrAnalyticsErr, "invalid key")
	ErrSortNotSupported = errors.Wrap(ErrAnalyticsErr, "sort parameter not supported")
)

type analyticsRepository struct {
	database db.Analytics
}

type AnalyticsRepository interface {
	QueryTopResultsByKey(ctx context.Context, params QueryTopResultsByKeyParams) (*QueryResults, error)
	QueryMostRecentResultsByKey(ctx context.Context, params QueryTopResultsByKeyParams) (*QueryResults, error)
	ParseAndStore(context.Context, uuid.UUID, string) error
	GetCountOverTime(ctx context.Context, params QueryTopResultsByKeyParams) (*CountOverTimeResults, error)
}

func NewAnalyticsRepository(d db.Analytics) AnalyticsRepository {
	return &analyticsRepository{
		database: d,
	}
}

type CountOverTime struct {
	Time  time.Time
	Count int64
}

type CountOverTimeResults struct {
	Results []CountOverTime
	Total   int64
}

func (r *analyticsRepository) GetCountOverTime(ctx context.Context, params QueryTopResultsByKeyParams) (*CountOverTimeResults, error) {
	if params.PageSize <= 0 {
		return nil, ErrInvalidPageSize
	}
	var results *CountOverTimeResults
	switch params.Key {
	case "node-results":
		dbParams := db.NumResultsOverTimeParams{
			PageSize:   int32(params.PageSize),
			PageNumber: int32(params.PageNumber),
		}
		log.Ctx(ctx).Trace().Interface("dbParams", dbParams).Msgf("querying db")
		rows, err := r.database.NumResultsOverTime(ctx, dbParams)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return nil, nil
		}
		r := make([]CountOverTime, len(rows))
		for i, row := range rows {
			r[i] = CountOverTime{
				Time:  row.TbTimestamp,
				Count: row.Count,
			}
		}
		results = &CountOverTimeResults{
			Results: r,
			Total:   rows[0].FullCount,
		}
	case "submissions":
		dbParams := db.NumSubmissionsOverTimeParams{
			PageSize:   int32(params.PageSize),
			PageNumber: int32(params.PageNumber),
		}
		log.Ctx(ctx).Trace().Interface("dbParams", dbParams).Msgf("querying db")
		rows, err := r.database.NumSubmissionsOverTime(ctx, dbParams)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return nil, nil
		}
		r := make([]CountOverTime, len(rows))
		for i, row := range rows {
			r[i] = CountOverTime{
				Time:  row.TbTimestamp,
				Count: row.Count,
			}
		}
		results = &CountOverTimeResults{
			Results: r,
			Total:   rows[0].FullCount,
		}
	default:
		return nil, ErrInvalidKey
	}
	return results, nil
}

func NewQueryTopResultsByKeyParams() QueryTopResultsByKeyParams {
	return QueryTopResultsByKeyParams{
		PageSize:   10,
		PageNumber: 1,
		Sort:       "-count",
	}
}

type QueryTopResultsByKeyParams struct {
	Key        string
	PageSize   int
	PageNumber int
	Sort       string
}

var sort_map = map[string]string{
	"count":           "count",
	"meta.count":      "count",
	"meta.created_at": "created_at",
}

type QueryResults struct {
	Results []QueryResult
	Total   int64
}

type QueryResult struct {
	Key   string
	Value interface{}
}

func (r *analyticsRepository) QueryTopResultsByKey(ctx context.Context, params QueryTopResultsByKeyParams) (*QueryResults, error) {
	reverse := strings.HasPrefix(params.Sort, "-")
	var ok bool
	params.Sort, ok = sort_map[strings.TrimPrefix(params.Sort, "-")]
	if !ok {
		return nil, ErrSortNotSupported
	}
	dbParams := db.QueryTopResultsByKeyParams{
		Key:        params.Key,
		PageSize:   int32(params.PageSize),
		PageNumber: int32(params.PageNumber),
		Sort:       params.Sort,
		Reverse:    reverse,
	}
	if dbParams.PageSize <= 0 {
		return nil, ErrInvalidPageSize
	}
	if dbParams.Key == "" {
		return nil, ErrInvalidKey
	}
	log.Ctx(ctx).Trace().Interface("dbParams", dbParams).Msgf("querying db")
	rows, err := r.database.QueryTopResultsByKey(ctx, dbParams)
	if err != nil {
		return nil, err
	}
	total, err := r.database.CountQueryTopResultsByKey(ctx, dbParams.Key)
	if err != nil {
		return nil, err
	}
	results := make([]QueryResult, len(rows))
	for i, row := range rows {
		results[i] = QueryResult{
			Key:   row.Value,
			Value: row.Count,
		}
	}
	return &QueryResults{
		Results: results,
		Total:   total,
	}, nil
}

func (r *analyticsRepository) QueryMostRecentResultsByKey(ctx context.Context, params QueryTopResultsByKeyParams) (*QueryResults, error) {
	reverse := strings.HasPrefix(params.Sort, "-")
	var ok bool
	params.Sort, ok = sort_map[strings.TrimPrefix(params.Sort, "-")]
	if !ok {
		return nil, ErrSortNotSupported
	}
	dbParams := db.QueryMostRecentResultsByKeyParams{
		Key:        params.Key,
		PageSize:   int32(params.PageSize),
		PageNumber: int32(params.PageNumber),
		Sort:       params.Sort,
		Reverse:    reverse,
	}
	if dbParams.PageSize <= 0 {
		return nil, ErrInvalidPageSize
	}
	if dbParams.Key == "" {
		return nil, ErrInvalidKey
	}
	log.Ctx(ctx).Trace().Interface("dbParams", dbParams).Msgf("querying db")
	rows, err := r.database.QueryMostRecentResultsByKey(ctx, dbParams)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &QueryResults{
			Results: []QueryResult{},
			Total:   0,
		}, nil
	}
	results := make([]QueryResult, len(rows))
	for i, row := range rows {
		results[i] = QueryResult{
			Key:   row.Value,
			Value: row.CreatedAt,
		}
	}
	return &QueryResults{
		Results: results,
		Total:   rows[0].FullCount,
	}, nil
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
