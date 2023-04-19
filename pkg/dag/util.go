package dag

import (
	"context"
	"time"
)

var (
	MaxTime = time.Unix(1<<63-62135596801, 999999999)
)

func GetDagStartTime[T any](ctx context.Context, dag []Node[T]) (time.Time, error) {
	earliestStartTime := MaxTime
	for _, d := range dag {
		n, err := d.Get(ctx)
		if err != nil {
			return time.Time{}, err
		}
		if n.Metadata.StartedAt.IsZero() { // Not started
			continue
		}
		if n.Metadata.StartedAt.Before(earliestStartTime) {
			earliestStartTime = n.Metadata.StartedAt
		}
	}
	if earliestStartTime == MaxTime {
		return time.Time{}, nil
	}
	return earliestStartTime, nil
}

func GetEndTimeIfDagComplete[T any](ctx context.Context, dag []Node[T]) (time.Time, error) {
	return recurseChildren(ctx, dag, time.Time{})
}

func recurseChildren[T any](ctx context.Context, dag []Node[T], currentEndTime time.Time) (time.Time, error) {
	if len(dag) == 0 {
		return currentEndTime, nil
	}
	for _, d := range dag {
		n, err := d.Get(ctx)
		if err != nil {
			return time.Time{}, err
		}
		if n.Metadata.EndedAt.IsZero() { // Still running
			return time.Time{}, nil
		}
		if n.Metadata.EndedAt.After(currentEndTime) {
			currentEndTime = n.Metadata.EndedAt
		}
		currentEndTime, err = recurseChildren(ctx, n.Children, currentEndTime)
		if err != nil {
			return time.Time{}, err
		}
	}
	return currentEndTime, nil
}
