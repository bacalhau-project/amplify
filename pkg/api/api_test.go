package api

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/bacalhau-project/amplify/pkg/item"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/google/uuid"
	"gotest.tools/assert"
)

func TestAPI_TemplatesSuccessfullyRender(t *testing.T) {
	w := httptest.NewRecorder()
	persistence := db.NewInMemDB()
	api, err := NewAmplifyAPI(&mockQueueRepository{}, task.NewMockTaskFactory(persistence))
	assert.NilError(t, err)
	tests := []struct {
		template string
		mockData interface{}
	}{
		{"graph.html.tmpl", GraphCollection{Data: []NodeSpec{}, Links: &PaginationLinks{}}},
		{"home.html.tmpl", Info{Links: mockLinks()}},
		{"job.html.tmpl", JobDatum{Data: &JobSpec{Id: "test", Attributes: &JobSpecAttributes{}, Links: mockLinks()}}},
		{"jobs.html.tmpl", JobCollection{Data: []JobSpec{}, Links: &PaginationLinks{}}},
		{"queue.html.tmpl", QueueCollection{Data: []QueueItem{}, Links: &PaginationLinks{}}},
		{"queueItem.html.tmpl", QueueDatum{Data: &QueueItemDetail{Id: "test", Meta: &QueueMetadata{}, Links: mockLinks()}}},
	}
	for _, tt := range tests {
		t.Run(tt.template, func(t *testing.T) {
			err := api.writeHTML(w, tt.template, tt.mockData)
			assert.NilError(t, err)
		})
	}

}

func mockLinks() *map[string]string {
	return &map[string]string{
		"self": "/self",
		"home": "/",
		"list": "/list",
	}
}

var _ item.QueueRepository = &mockQueueRepository{}

type mockQueueRepository struct{}

func (*mockQueueRepository) Count(context.Context) (int64, error) {
	return 0, nil
}

func (*mockQueueRepository) Create(context.Context, item.ItemParams) error {
	return nil
}

func (*mockQueueRepository) Get(context.Context, uuid.UUID) (*item.Item, error) {
	return nil, nil
}

func (*mockQueueRepository) List(context.Context, item.ListParams) ([]*item.Item, error) {
	return nil, nil
}
