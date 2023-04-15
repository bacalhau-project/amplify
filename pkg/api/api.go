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
	"sync"
	"time"

	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/bacalhau-project/amplify/pkg/item"
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

type amplifyAPI struct {
	*sync.Mutex
	er   item.QueueRepository
	tf   task.TaskFactory
	tmpl *template.Template
}

func NewAmplifyAPI(er item.QueueRepository, tf task.TaskFactory) (*amplifyAPI, error) {
	tmpl := template.New("master").Funcs(funcs)
	tmpl, err := tmpl.ParseFS(content, "templates/*.html.tmpl")
	if err != nil {
		return nil, err
	}
	return &amplifyAPI{
		er:   er,
		tf:   tf,
		tmpl: tmpl,
	}, nil
}

// Amplify Home
// (GET /)
func (a *amplifyAPI) Get(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("Get")
	_, err := w.Write([]byte(
		`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Amplify</title>
</head>
<body>
<h1>Amplify</h1>
<p>Amplify enhances, enriches, and explains your data, automatically.</p>
<p>Find out more on the <a href="https://github.com/bacalhau-project/amplify">repository homepage</a>.</p>
<p>This is a temporary home page until the pretty UI is ready.</p>
<h2>API</h2>
<p>You can view the Amplify API by browsing to <a href="/api/v0">/api/v0</a>.</p>
</body>
</html>`,
	))
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("Could not write response")
	}
}

// Amplify home
// (GET /v0)
func (a *amplifyAPI) GetApiV0(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("GetApiV0")
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
func (a *amplifyAPI) GetApiV0Jobs(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("GetApiV0Jobs")
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
func (a *amplifyAPI) GetApiV0JobsId(w http.ResponseWriter, r *http.Request, id string) {
	log.Ctx(r.Context()).Trace().Str("id", id).Msg("GetApiV0JobsId")
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
func (a *amplifyAPI) GetApiV0Queue(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("GetApiV0Queue")
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
func (a *amplifyAPI) GetApiV0QueueId(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	log.Ctx(r.Context()).Trace().Str("id", id.String()).Msg("GetApiV0QueueId")
	e, err := a.getItemDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
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
func (a *amplifyAPI) PutApiV0QueueId(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	log.Ctx(r.Context()).Trace().Str("id", id.String()).Msg("PutV0QueueId")
	var body ExecutionRequest
	switch r.Header.Get("Content-type") {
	case "application/json":
		fallthrough
	case "application/vnd.api+json":
		// Parse request body
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			sendError(r.Context(), w, http.StatusBadRequest, "Could not parse request body", err.Error())
			return
		}
	default:
		sendError(r.Context(), w, http.StatusBadRequest, "Wrong content type", "The Content-Type header is not set according to the API spec.")
		return
	}

	err := a.CreateExecution(r.Context(), id, body.Cid)
	if err != nil {
		sendError(r.Context(), w, http.StatusInternalServerError, "Could not create execution", err.Error())
		return
	}

	w.WriteHeader(202)
}

// Run all workflows for a CID (not recommended)
// (POST /api/v0/queue)
func (a *amplifyAPI) PostApiV0Queue(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("PostApiV0Queue")
	var body ExecutionRequest
	switch r.Header.Get("Content-type") {
	case "application/x-www-form-urlencoded":
		err := r.ParseForm()
		if err != nil {
			sendError(r.Context(), w, http.StatusBadRequest, "Could not parse request body", err.Error())
			return
		}
		body.Cid = r.FormValue("cid")
	default:
		sendError(r.Context(), w, http.StatusBadRequest, "Wrong content type", "The Content-Type header is not set according to the API spec.")
		return
	}

	err := a.CreateExecution(r.Context(), uuid.New(), body.Cid)
	if err != nil {
		sendError(r.Context(), w, http.StatusInternalServerError, "Could not create execution", err.Error())
		return
	}

	w.WriteHeader(202)
}

func (a *amplifyAPI) CreateExecution(ctx context.Context, executionID uuid.UUID, cid string) error {
	return a.er.Create(ctx, item.ItemParams{
		ID:  executionID,
		CID: cid,
	})
}

// Get Amplify work graph
// (GET /v0/graph)
func (a *amplifyAPI) GetApiV0Graph(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("GetApiV0Workflows")
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
				OutputId:  &input.OutputID,
				Path:      &input.Path,
				Root:      &input.Root,
				StepId:    &input.NodeID,
				Predicate: &input.Predicate,
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
	items := make([]Item, len(e))
	for i, item := range e {
		items[i] = *buildItem(ctx, item)
	}
	return &Queue{
		Data: &items,
		Links: &Links{
			"self": "/api/v0/queue",
			"home": "/api/v0",
		},
	}, nil
}

func makeNode(ctx context.Context, dagNode dag.Node[dag.IOSpec], rootId openapi_types.UUID) Node {
	child, err := dagNode.Get(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not get node")
		return Node{}
	}
	inputs := make([]ExecutionRequest, len(child.Inputs))
	for idx, input := range child.Inputs {
		inputs[idx] = ExecutionRequest{
			Cid: input.CID(),
		}
	}
	outputs := make([]ExecutionRequest, len(child.Outputs))
	for idx, output := range child.Outputs {
		outputs[idx] = ExecutionRequest{
			Cid: output.CID(),
		}
	}
	node := Node{
		Id:      rootId,
		Name:    util.StrP(child.Name),
		Inputs:  inputs,
		Outputs: outputs,
		Result: &ItemResult{
			Id:      util.StrP(child.Results.ID),
			Stderr:  util.StrP(child.Results.StdErr),
			Stdout:  util.StrP(child.Results.StdOut),
			Skipped: util.BoolP(child.Results.Skipped),
		},
		Links: &Links{
			"self": fmt.Sprintf("/api/v0/queue/%s", rootId),
			"list": "/api/v0/queue",
		},
		Metadata: ItemMetadata{
			Submitted: child.Metadata.CreatedAt.Format(time.RFC3339),
			Status:    child.Metadata.Status,
		},
	}
	if !child.Metadata.StartedAt.IsZero() {
		node.Metadata.Started = util.StrP(child.Metadata.StartedAt.Format(time.RFC3339))
	}
	if !child.Metadata.EndedAt.IsZero() {
		node.Metadata.Ended = util.StrP(child.Metadata.EndedAt.Format(time.RFC3339))
	}
	if len(child.Children) > 0 {
		children := make([]Node, len(child.Children))
		for idx, c := range child.Children {
			children[idx] = makeNode(ctx, c, rootId)
		}
		node.Children = &children
	}
	return node
}

func (a *amplifyAPI) getItemDetail(ctx context.Context, executionId openapi_types.UUID) (*Node, error) {
	i, err := a.er.Get(ctx, executionId)
	if err != nil {
		return nil, err
	}
	dag := make([]Node, len(i.RootNodes))
	for idx, child := range i.RootNodes {
		dag[idx] = makeNode(ctx, child, executionId)
	}
	results := GetLeafOutputs(ctx, i.RootNodes)
	results = dedup(results)
	outputs := make([]ExecutionRequest, len(results))
	for idx, output := range results {
		outputs[idx] = ExecutionRequest{
			Cid: output,
		}
	}

	v := &Node{
		Id:   i.ID,
		Type: "node",
		Inputs: []ExecutionRequest{{
			Cid: i.CID,
		}},
		Outputs: outputs,
		Metadata: ItemMetadata{
			Submitted: i.Metadata.CreatedAt.Format(time.RFC3339),
			Status:    childStatusesToList(ctx, i.RootNodes),
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

func parseChildrenStatus(ctx context.Context, children []dag.Node[dag.IOSpec], statuses map[int32]string) map[int32]string {
	for _, child := range children {
		_, ok := statuses[child.ID()]
		if ok {
			continue
		}
		c, err := child.Get(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Could not get node")
			continue
		}
		statuses[child.ID()] = c.Metadata.Status
		if len(c.Children) > 0 {
			statuses = parseChildrenStatus(ctx, c.Children, statuses)
		}
	}
	return statuses
}

func childStatusesToList(ctx context.Context, children []dag.Node[dag.IOSpec]) string {
	statuses := parseChildrenStatus(ctx, children, make(map[int32]string, len(children)))
	jobsCompleted := 0
	for _, v := range statuses {
		if v == dag.Finished.String() {
			jobsCompleted++
		}
	}
	return fmt.Sprintf("%.0f%%", 100*float32(jobsCompleted)/float32(len(statuses)))
}

func buildItem(ctx context.Context, i *item.Item) *Item {
	v := Item{
		Id:   i.ID.String(),
		Type: "item",
		Metadata: ItemMetadata{
			Submitted: i.Metadata.CreatedAt.Format(time.RFC3339),
			Started:   util.StrP(i.Metadata.StartedAt.Format(time.RFC3339)),
			Ended:     util.StrP(i.Metadata.EndedAt.Format(time.RFC3339)),
			Status:    childStatusesToList(ctx, i.RootNodes),
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
func GetLeafOutputs(ctx context.Context, dag []dag.Node[dag.IOSpec]) []string {
	var results []string
	for _, child := range dag {
		c, err := child.Get(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Could not get node")
			return nil
		}
		if len(c.Children) == 0 {
			for _, output := range c.Outputs {
				results = append(results, output.CID())
			}
		} else {
			results = append(results, GetLeafOutputs(ctx, c.Children)...)
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
