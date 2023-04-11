package queue

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"sync"
	"time"

	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
)

// migrations holds the database migrations
//
//go:embed migrations/*
var embedFS embed.FS

type postgresQueueRepository struct {
	*sync.Mutex
	queries *db.Queries
	queue   Queue
	store   map[string]*Item // Temporary store to retain DAG
}

func NewPostgresQueueRepository(connStr string, queue Queue) (QueueRepository, error) {
	pdb, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	list, err := fs.Glob(embedFS, "migrations/*.sql")
	if err != nil {
		return nil, err
	}
	migrations := &migrate.MemoryMigrationSource{}
	for _, f := range list {
		b, err := fs.ReadFile(embedFS, f)
		if err != nil {
			return nil, err
		}
		m, err := migrate.ParseMigration(f, bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		migrations.Migrations = append(migrations.Migrations, m)
	}
	migrationCount, err := migrate.Exec(pdb, "postgres", migrations, migrate.Up)
	if err != nil {
		return nil, err
	}
	log.Debug().Int("migrations", migrationCount).Msg("migrations applied")
	queries := db.New(pdb)
	return &postgresQueueRepository{
		Mutex:   &sync.Mutex{},
		queries: queries,
		queue:   queue,
		store:   make(map[string]*Item),
	}, nil
}

func (r *postgresQueueRepository) Create(ctx context.Context, req Item) error {
	r.Lock()
	defer r.Unlock()
	if req.ID == "" {
		return ErrItemNoID
	}
	id, err := uuid.Parse(req.ID)
	if err != nil {
		return err
	}
	_, err = r.queries.GetQueueItemDetail(ctx, id)
	if err == nil {
		return ErrAlreadyExists
	}

	err = r.queries.CreateQueueItem(ctx, db.CreateQueueItemParams{
		ID:        id,
		Inputs:    []string{req.CID},
		CreatedAt: req.Metadata.CreatedAt,
	})
	if err != nil {
		return err
	}
	r.store[req.ID] = &req
	for _, d := range req.Dag {
		err := r.queue.Enqueue(d.Execute)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *postgresQueueRepository) Get(ctx context.Context, idStr string) (*Item, error) {
	r.Lock()
	defer r.Unlock()
	if idStr == "" {
		return nil, ErrItemNoID
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}

	// TODO: temp. If contained within the in-memory store, return it
	i, ok := r.store[idStr]
	if ok {
		r.updateStartStopTime(idStr)
		return i, nil
	}

	// If not, check the persisted store
	item, err := r.queries.GetQueueItemDetail(ctx, id)
	if err != nil {
		return nil, err
	}
	log.Debug().Interface("item", item).Msg("item")
	return deserializeDBItem(item), nil
}

func (r *postgresQueueRepository) List(ctx context.Context) ([]*Item, error) {
	r.Lock()
	defer r.Unlock()
	dbItems, err := r.queries.ListQueueItems(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]*Item, 0, len(r.store)+len(dbItems))
	for _, i := range r.store {
		r.updateStartStopTime(i.ID)
	}
	for _, i := range r.store {
		list = append(list, i)
	}
	for _, i := range dbItems {
		_, ok := r.store[i.ID.String()]
		if !ok {
			list = append(list, deserializeDBItem(i))
		}
	}
	return list, nil
}

// TODO: This is really bad. We should use a channel to set this
func (r *postgresQueueRepository) updateStartStopTime(id string) {
	i := r.store[id]
	if i.Metadata.StartedAt.IsZero() {
		for _, d := range i.Dag {
			if !d.Meta().StartedAt.IsZero() {
				i.Metadata.StartedAt = d.Meta().StartedAt
				break
			}
		}
	}
	if i.Metadata.EndedAt.IsZero() {
		// All dags must have finished
		var t time.Time
		ok := true
		for _, d := range i.Dag {
			finTime := recurseLastTime(d)
			if finTime.IsZero() {
				ok = false
				break
			}
			if finTime.After(t) {
				t = finTime
			}
		}
		if ok {
			i.Metadata.EndedAt = t
		}
	}
}

func deserializeDBItem(dbItem db.QueueItem) *Item {
	return &Item{
		ID: dbItem.ID.String(),
		Metadata: ItemMetadata{
			CreatedAt: dbItem.CreatedAt,
		},
		CID: dbItem.Inputs[0],
	}
}
