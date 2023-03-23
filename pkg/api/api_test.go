package api

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/task"
	"gotest.tools/assert"
)

func TestAPI_TemplatesSuccessfullyRender(t *testing.T) {
	w := httptest.NewRecorder()
	api := NewAmplifyAPI(&mockQueueRepository{}, &task.TaskFactory{})
	tests := []struct {
		template string
		mockData interface{}
	}{
		{"graph.html.tmpl", Graph{Data: &[]NodeConfig{}, Links: mockLinks()}},
		{"home.html.tmpl", Home{Links: mockLinks()}},
		{"job.html.tmpl", Job{Links: mockLinks()}},
		{"jobs.html.tmpl", Jobs{Data: &[]Job{}, Links: mockLinks()}},
		{"queue.html.tmpl", Queue{Data: &[]Item{}, Links: mockLinks()}},
		{"queueItem.html.tmpl", Node{Inputs: []ExecutionRequest{}, Children: &[]Node{}, Links: mockLinks()}},
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
		"home": "/",
		"list": "/list",
	}
}

var _ queue.QueueRepository = &mockQueueRepository{}

type mockQueueRepository struct{}

func (*mockQueueRepository) Create(context.Context, queue.Item) error {
	return nil
}

func (*mockQueueRepository) Get(context.Context, string) (*queue.Item, error) {
	return nil, nil
}

func (*mockQueueRepository) List(context.Context) ([]*queue.Item, error) {
	return nil, nil
}
