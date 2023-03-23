package api

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/bacalhau-project/amplify/pkg/util"
	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var (
	funcs = template.FuncMap{"marshall": func(v interface{}) string {
		b, _ := json.Marshal(v)
		return string(b)
	}}
)

// content holds our static web server content.
//
//go:embed templates/*
var content embed.FS

var _ ServerInterface = (*amplifyAPI)(nil)

type amplifyAPI struct {
	*sync.Mutex
	er   queue.QueueRepository
	tf   *task.TaskFactory
	tmpl *template.Template
}

// TODO: Getting gross
func NewAmplifyAPI(er queue.QueueRepository, tf *task.TaskFactory) *amplifyAPI {
	tmpl := template.New("master").Funcs(funcs)
	tmpl, err := tmpl.ParseFS(content, "templates/*.html.tmpl")
	if err != nil {
		panic(err)
	}
	return &amplifyAPI{
		er:   er,
		tf:   tf,
		tmpl: tmpl,
	}
}

// Amplify home
// (GET /v0)
func (a *amplifyAPI) GetV0(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("GetV0")
	home := &Home{
		Type: util.StrP("home"),
		Links: util.MapP(map[string]interface{}{
			"self":  "/api/v0",
			"queue": "/api/v0/queue",
			"jobs":  "/api/v0/jobs",
			"graph": "/api/v0/graph",
		}),
	}
	switch r.Header.Get("Content-type") {
	case "application/json":
		fallthrough
	case "application/vnd.api+json":
		w.Header().Set("Content-Type", "application/vnd.api+json")
		err := json.NewEncoder(w).Encode(home)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render JSON", err.Error())
			return
		}
	default:
		err := a.writeHTML(w, "home.html.tmpl", home)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render HTML", err.Error())
			return
		}
	}
}

// List all Amplify jobs
// (GET /v0/jobs)
func (a *amplifyAPI) GetV0Jobs(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("GetV0Jobs")
	jobs, err := a.getJobs()
	if err != nil {
		sendError(r.Context(), w, http.StatusInternalServerError, "Could not get jobs", err.Error())
		return
	}
	switch r.Header.Get("Content-type") {
	case "application/json":
		fallthrough
	case "application/vnd.api+json":
		w.Header().Set("Content-Type", "application/vnd.api+json")
		err := json.NewEncoder(w).Encode(jobs)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render JSON", err.Error())
			return
		}
	default:
		err := a.writeHTML(w, "jobs.html.tmpl", jobs)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render HTML", err.Error())
			return
		}
	}
}

// Get a job by id
// (GET /v0/jobs/{id})
func (a *amplifyAPI) GetV0JobsId(w http.ResponseWriter, r *http.Request, id string) {
	log.Ctx(r.Context()).Trace().Str("id", id).Msg("GetV0JobsId")
	j, err := a.getJob(id)
	if err != nil {
		if errors.Is(err, task.ErrJobNotFound) {
			sendError(r.Context(), w, http.StatusNotFound, "Job not found", fmt.Sprintf("Job %s not found", id))
		} else {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not get job", err.Error())
		}
		return
	}
	switch r.Header.Get("Content-type") {
	case "application/json":
		fallthrough
	case "application/vnd.api+json":
		w.Header().Set("Content-Type", "application/vnd.api+json")
		err := json.NewEncoder(w).Encode(j)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render JSON", err.Error())
			return
		}
	default:
		err := a.writeHTML(w, "job.html.tmpl", j)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render HTML", err.Error())
			return
		}
	}
}

// Amplify work queue
// (GET /v0/queue)
func (a *amplifyAPI) GetV0Queue(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("GetV0Queue")
	executions, err := a.getQueue(r.Context())
	if err != nil {
		sendError(r.Context(), w, http.StatusInternalServerError, "Could not get executions", err.Error())
		return
	}
	switch r.Header.Get("Content-type") {
	case "application/json":
		fallthrough
	case "application/vnd.api+json":
		w.Header().Set("Content-Type", "application/vnd.api+json")
		err := json.NewEncoder(w).Encode(executions)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render JSON", err.Error())
			return
		}
	default:
		err := a.writeHTML(w, "queue.html.tmpl", executions)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render HTML", err.Error())
			return
		}
	}
}

// Get an item from the queue by id
// (GET /v0/queue/{id})
func (a *amplifyAPI) GetV0QueueId(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	log.Ctx(r.Context()).Trace().Str("id", id.String()).Msg("GetV0QueueId")
	e, err := a.getItemDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, queue.ErrNotFound) {
			sendError(r.Context(), w, http.StatusNotFound, "Execution not found", fmt.Sprintf("Execution %s not found", id.String()))
		} else {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not get execution", err.Error())
		}
		return
	}
	switch r.Header.Get("Content-type") {
	case "application/json":
		fallthrough
	case "application/vnd.api+json":
		w.Header().Set("Content-Type", "application/vnd.api+json")
		err := json.NewEncoder(w).Encode(e)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render JSON", err.Error())
			return
		}
	default:
		err := a.writeHTML(w, "queueItem.html.tmpl", e)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render HTML", err.Error())
			return
		}
	}
}

// Run all workflows for a CID
// (PUT /v0/queue/{id})
func (a *amplifyAPI) PutV0QueueId(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	log.Ctx(r.Context()).Trace().Str("id", id.String()).Msg("PutV0QueueId")
	// Parse request body
	var body ExecutionRequest
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		sendError(r.Context(), w, http.StatusBadRequest, "Could not parse request body", err.Error())
		return
	}
	task, err := a.tf.CreateTask(r.Context(), "", body.Cid)
	if err != nil {
		sendError(r.Context(), w, http.StatusInternalServerError, "Could not create task", err.Error())
		return
	}
	err = a.er.Create(r.Context(), queue.Item{
		ID:  id.String(),
		CID: body.Cid,
		Dag: []*dag.Node[dag.IOSpec]{task},
		Metadata: queue.ItemMetadata{
			CreatedAt: time.Now(),
		},
	})
	if err != nil {
		sendError(r.Context(), w, http.StatusInternalServerError, "Could not create execution", err.Error())
		return
	}
	w.WriteHeader(202)
}

// Get Amplify work graph
// (GET /v0/graph)
func (a *amplifyAPI) GetV0Graph(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("GetV0Workflows")
	nn := a.tf.NodeNames()
	graph := make([]NodeConfig, len(nn))
	for idx, n := range nn {
		node, err := a.tf.GetNode(n)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not get node", err.Error())
			return
		}
		inputs := make([]NodeInput, len(node.Inputs))
		for i, input := range node.Inputs {
			inputs[i] = NodeInput{
				OutputId: &input.OutputID,
				Path:     &input.Path,
				Root:     &input.Root,
				StepId:   &input.NodeID,
			}
		}
		outputs := make([]NodeOutput, len(node.Outputs))
		for i, output := range node.Outputs {
			outputs[i] = NodeOutput{
				Id:   &output.ID,
				Path: &output.Path,
			}
		}
		graph[idx] = NodeConfig{
			Id:      &node.ID,
			Inputs:  &inputs,
			JobId:   &node.JobID,
			Outputs: &outputs,
		}
	}
	outputGraph := Graph{
		Data: &graph,
		Links: &Links{
			"self": "/api/v0/graph",
			"home": "/api/v0",
		},
	}
	switch r.Header.Get("Content-type") {
	case "application/json":
		fallthrough
	case "application/vnd.api+json":
		w.Header().Set("Content-Type", "application/vnd.api+json")
		err := json.NewEncoder(w).Encode(outputGraph)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render JSON", err.Error())
			return
		}
	default:
		err := a.writeHTML(w, "graph.html.tmpl", outputGraph)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render HTML", err.Error())
			return
		}
	}
}

func (a *amplifyAPI) getJobs() (*Jobs, error) {
	jobList := make([]Job, len(a.tf.JobNames()))
	for i, id := range a.tf.JobNames() {
		j, err := a.getJob(id)
		if err != nil {
			return nil, err
		}
		jobList[i] = *j
	}
	return &Jobs{
		Data: &jobList,
		Links: &Links{
			"self": "/api/v0/jobs",
			"home": "/api/v0",
		},
	}, nil
}

func (a *amplifyAPI) getJob(jobId string) (*Job, error) {
	j, err := a.tf.GetJob(jobId)
	if err != nil {
		return nil, err
	}
	return &Job{
		Type:       "job",
		Image:      j.Image,
		Entrypoint: &j.Entrypoint,
		Id:         j.ID,
		Links: &Links{
			"self": fmt.Sprintf("/api/v0/jobs/%s", j.ID),
			"list": "/api/v0/jobs",
			"home": "/api/v0",
		},
	}, nil
}

func (a *amplifyAPI) getQueue(ctx context.Context) (*Queue, error) {
	e, err := a.er.List(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(e, func(i, j int) bool {
		return e[i].Metadata.CreatedAt.UnixNano() < e[j].Metadata.CreatedAt.UnixNano()
	})
	items := make([]Item, len(e))
	for i, item := range e {
		items[i] = *buildItem(item)
	}
	return &Queue{
		Data: &items,
		Links: &Links{
			"self": "/api/v0/queue",
			"home": "/api/v0",
		},
	}, nil
}

func makeNode(child *dag.Node[dag.IOSpec], rootId openapi_types.UUID) Node {
	inputs := make([]ExecutionRequest, len(child.Inputs()))
	for idx, input := range child.Inputs() {
		inputs[idx] = ExecutionRequest{
			Cid: input.CID(),
		}
	}
	outputs := make([]ExecutionRequest, len(child.Outputs()))
	for idx, output := range child.Outputs() {
		outputs[idx] = ExecutionRequest{
			Cid: output.CID(),
		}
	}
	node := Node{
		Id:      rootId,
		Inputs:  inputs,
		Outputs: outputs,
		Execution: &ExecutionInfo{
			Id:     util.StrP(child.Status().ID),
			Status: util.StrP(child.Status().Status),
			Stderr: util.StrP(child.Status().StdErr),
			Stdout: util.StrP(child.Status().StdOut),
		},
		Links: &Links{
			"self": fmt.Sprintf("/api/v0/queue/%s", rootId),
			"list": "/api/v0/queue",
		},
		Metadata: ItemMetadata{
			Submitted: child.Meta().CreatedAt.Format(time.RFC3339),
		},
	}
	if !child.Meta().StartedAt.IsZero() {
		node.Metadata.Started = util.StrP(child.Meta().StartedAt.Format(time.RFC3339))
	}
	if !child.Meta().EndedAt.IsZero() {
		node.Metadata.Ended = util.StrP(child.Meta().EndedAt.Format(time.RFC3339))
	}
	if len(child.Children()) > 0 {
		children := make([]Node, len(child.Children()))
		for idx, c := range child.Children() {
			children[idx] = makeNode(c, rootId)
		}
		node.Children = &children
	}
	return node
}

func (a *amplifyAPI) getItemDetail(ctx context.Context, executionId openapi_types.UUID) (*Node, error) {
	i, err := a.er.Get(ctx, executionId.String())
	if err != nil {
		return nil, err
	}
	dag := make([]Node, len(i.Dag))
	for idx, child := range i.Dag {
		dag[idx] = makeNode(child, executionId)
	}
	results := getLeafOutputs(i.Dag)
	results = dedup(results)
	outputs := make([]ExecutionRequest, len(results))
	for idx, output := range results {
		outputs[idx] = ExecutionRequest{
			Cid: output,
		}
	}

	v := &Node{
		Id:   uuid.MustParse(i.ID),
		Type: "node",
		Inputs: []ExecutionRequest{{
			Cid: i.CID,
		}},
		Outputs: outputs,
		Metadata: ItemMetadata{
			Submitted: i.Metadata.CreatedAt.Format(time.RFC3339),
		},
		Links: &Links{
			"self": fmt.Sprintf("/api/v0/queue/%s", i.ID),
			"list": "/api/v0/queue",
		},
		Children: &dag,
	}
	if !i.Metadata.StartedAt.IsZero() {
		v.Metadata.Started = util.StrP(i.Metadata.StartedAt.Format(time.RFC3339))
	}
	if !i.Metadata.EndedAt.IsZero() {
		v.Metadata.Ended = util.StrP(i.Metadata.EndedAt.Format(time.RFC3339))
	}
	return v, nil
}

func buildItem(i *queue.Item) *Item {
	v := Item{
		Id:   i.ID,
		Type: "item",
		Metadata: ItemMetadata{
			Submitted: i.Metadata.CreatedAt.Format(time.RFC3339),
		},
		Links: &Links{
			"self": fmt.Sprintf("/api/v0/queue/%s", i.ID),
			"list": "/api/v0/queue",
		},
	}
	if !i.Metadata.StartedAt.IsZero() {
		v.Metadata.Started = util.StrP(i.Metadata.StartedAt.Format(time.RFC3339))
	}
	if !i.Metadata.EndedAt.IsZero() {
		v.Metadata.Ended = util.StrP(i.Metadata.EndedAt.Format(time.RFC3339))
	}
	return &v
}

func (a *amplifyAPI) writeHTML(w http.ResponseWriter, templateName string, data interface{}) error {
	t := a.tmpl.Lookup(templateName).Option("missingkey=error")
	buf := &bytes.Buffer{}
	err := t.Execute(buf, data)
	if err != nil {
		return err
	} else {
		w.Header().Set("Content-Type", "text/html")
		_, err := buf.WriteTo(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func sendError(ctx context.Context, w http.ResponseWriter, statusCode int, userErr, devErr string) {
	log.Ctx(ctx).Warn().Int("status", statusCode).Str("user", userErr).Str("dev", devErr).Msg("API Error")
	e := Error{
		Title:  &userErr,
		Detail: &devErr,
	}
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(&Errors{e})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error sending error response")
	}
}

// getLeafOutputs returns the outputs of the leaf nodes in the DAG
func getLeafOutputs(dag []*dag.Node[dag.IOSpec]) []string {
	var results []string
	for _, child := range dag {
		if len(child.Children()) == 0 {
			for _, output := range child.Outputs() {
				results = append(results, output.CID())
			}
		} else {
			results = append(results, getLeafOutputs(child.Children())...)
		}
	}
	return results
}

// dedup removes duplicate strings from a slice
func dedup(s []string) []string {
	m := make(map[string]bool)
	for _, v := range s {
		m[v] = true
	}
	var results []string
	for k := range m {
		results = append(results, k)
	}
	return results
}
