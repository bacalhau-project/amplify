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
		{"graph.html.tmpl", Graph{Data: &[]NodeConfig{}, Links: mockLinks()}},
		{"home.html.tmpl", Home{Links: mockLinks()}},
		{"job.html.tmpl", Job{Links: mockLinks()}},
		{"jobs.html.tmpl", Jobs{Data: &[]Job{}, Links: mockLinks()}},
		{"queue.html.tmpl", Queue{Data: &[]Item{{Links: mockLinks()}}, Links: mockLinks()}},
		{"queueItem.html.tmpl", Node{Inputs: []ExecutionRequest{}, Children: &[]Node{{Links: mockLinks()}}, Links: mockLinks()}},
	}
	for _, tt := range tests {
		t.Run(tt.template, func(t *testing.T) {
			err := api.writeHTML(w, tt.template, tt.mockData)
			assert.NilError(t, err)
		})
	}

}

func mockLinks() *Links {
	return &Links{
		"self": "/self",
		"home": "/",
		"list": "/list",
	}
}

var _ item.QueueRepository = &mockQueueRepository{}

type mockQueueRepository struct{}

func (*mockQueueRepository) Create(context.Context, item.ItemParams) error {
	return nil
}

func (*mockQueueRepository) Get(context.Context, uuid.UUID) (*item.Item, error) {
	return nil, nil
}

func (*mockQueueRepository) List(context.Context) ([]*item.Item, error) {
	return nil, nil
}
