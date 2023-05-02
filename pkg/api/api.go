package api

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/bacalhau-project/amplify/pkg/analytics"
	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/bacalhau-project/amplify/pkg/item"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/bacalhau-project/amplify/pkg/util"
	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

var (
	funcs = template.FuncMap{"marshall": func(v interface{}) string {
		b, _ := json.Marshal(v)
		return string(b)
	}}
	ErrPageOutOfRange = fmt.Errorf("page out of range")
)

// content holds our static web server content.
//
//go:embed templates/*
var content embed.FS

type amplifyAPI struct {
	*sync.Mutex
	er        item.QueueRepository
	tf        task.TaskFactory
	tmpl      *template.Template
	analytics analytics.AnalyticsRepository
}

var _ ServerInterface = (*amplifyAPI)(nil)

func NewAmplifyAPI(er item.QueueRepository, tf task.TaskFactory, analytics analytics.AnalyticsRepository) (*amplifyAPI, error) {
	tmpl := template.New("master").Funcs(funcs)
	tmpl, err := tmpl.ParseFS(content, "templates/*.html.tmpl")
	if err != nil {
		return nil, err
	}
	return &amplifyAPI{
		er:        er,
		tf:        tf,
		tmpl:      tmpl,
		analytics: analytics,
	}, nil
}

// GetV0AnalyticsResultsResultMetadataKey implements ServerInterface
func (a *amplifyAPI) GetV0AnalyticsResultsResultMetadataKey(w http.ResponseWriter, r *http.Request, resultMetadataKey string, params GetV0AnalyticsResultsResultMetadataKeyParams) {
	log.Ctx(r.Context()).Trace().Str("key", resultMetadataKey).Msg("GetV0AnalyticsResultsResultMetadataKey")
	if params.PageSize == nil {
		params.PageSize = util.Int32P(10)
	}
	results, err := a.analytics.QueryTopResultsByKey(r.Context(), analytics.QueryTopResultsByKeyParams{
		Key:      resultMetadataKey,
		PageSize: int(*params.PageSize),
	})
	if err != nil {
		if errors.Is(err, analytics.ErrAnalyticsErr) {
			sendError(r.Context(), w, http.StatusBadRequest, "Could not query analytics", err.Error())
			return
		}
		sendError(r.Context(), w, http.StatusInternalServerError, "Could not query analytics", err.Error())
		return
	}
	resultDatum := make([]ResultDatum, len(results))
	index := 0
	for k, v := range results {
		resultDatum[index] = ResultDatum{
			Type: "ResultDatum",
			Id:   k,
			Meta: &map[string]interface{}{
				"count": v,
			},
		}
		index++
	}
	slices.SortFunc(resultDatum, func(i, j ResultDatum) bool {
		return (*i.Meta)["count"].(int64) > (*j.Meta)["count"].(int64)
	})
	response := &ResultCollection{
		Data: resultDatum,
		Links: &PaginationLinks{
			AdditionalProperties: map[string]string{
				"self":      "/api/v0/analytics/results/" + resultMetadataKey,
				"analytics": "/api/v0/analytics",
			},
		},
		Meta: &map[string]interface{}{
			"count": len(resultDatum),
		},
	}
	a.renderResponse(w, r, response, "results.html.tmpl")
}

// Amplify home
// (GET /v0)
func (a *amplifyAPI) GetV0(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("GetV0")
	home := &Info{
		Jsonapi: &Jsonapi{
			Version: util.StrP("1.1"),
		},
		Links: &map[string]string{
			"self":  "/api/v0",
			"queue": "/api/v0/queue",
			"jobs":  "/api/v0/jobs",
			"graph": "/api/v0/graph",
		},
	}
	a.renderResponse(w, r, home, "home.html.tmpl")
}

// List all Amplify jobs
// (GET /v0/jobs)
func (a *amplifyAPI) GetV0Jobs(w http.ResponseWriter, r *http.Request, params GetV0JobsParams) {
	log.Ctx(r.Context()).Trace().Msg("GetV0Jobs")
	paginationParams := parsePaginationParams(params.PageSize, params.PageNumber)
	jobList := make([]JobSpec, len(a.tf.JobNames()))
	for i, id := range a.tf.JobNames() {
		j, err := a.getJob(id)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not get jobs", err.Error())
			return
		}
		jobList[i] = *j
	}
	totalPages := 1
	if paginationParams.PageNumber > totalPages || paginationParams.PageNumber < 1 {
		sendError(r.Context(), w, http.StatusBadRequest, "Page out of range", ErrPageOutOfRange.Error())
		return
	}
	jobs := &JobCollection{
		Meta: &map[string]interface{}{
			"totalPages": &totalPages,
			"count":      len(jobList),
		},
		Data: jobList,
		Links: &PaginationLinks{
			First: util.StrP(fmt.Sprintf("/api/v0/jobs?page[size]=%d&page[number]=%d", paginationParams.PageSize, 1)),
			Last:  util.StrP(fmt.Sprintf("/api/v0/jobs?page[size]=%d&page[number]=%d", paginationParams.PageSize, totalPages)),
			Next:  util.StrP(fmt.Sprintf("/api/v0/jobs?page[size]=%d&page[number]=%d", paginationParams.PageSize, paginationParams.PageNumber+1)),
			Prev:  util.StrP(fmt.Sprintf("/api/v0/jobs?page[size]=%d&page[number]=%d", paginationParams.PageSize, paginationParams.PageNumber-1)),
			AdditionalProperties: map[string]Link{
				"self": "/api/v0/jobs",
				"home": "/api/v0",
			},
		},
	}
	a.renderResponse(w, r, jobs, "jobs.html.tmpl")
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
	jobDatum := &JobDatum{
		Data: j,
	}
	a.renderResponse(w, r, jobDatum, "job.html.tmpl")
}

func parsePaginationParams(pageSize *int32, pageNumber *int32) item.ListParams {
	paginationParams := item.NewListParams() // Use defaults lower down
	if pageSize != nil {
		paginationParams.PageSize = int(*pageSize)
	}
	if pageNumber != nil {
		paginationParams.PageNumber = int(*pageNumber)
	}
	return paginationParams
}

func parseGetQueueParams(params GetV0QueueParams) item.ListParams {
	paginationParams := item.NewListParams() // Use defaults lower down
	if params.PageSize != nil {
		paginationParams.PageSize = int(*params.PageSize)
	}
	if params.PageNumber != nil {
		paginationParams.PageNumber = int(*params.PageNumber)
	}
	if params.Sort != nil {
		paginationParams.Sort = *params.Sort
	}
	return paginationParams
}

// Amplify work queue
// (GET /v0/queue)
func (a *amplifyAPI) GetV0Queue(w http.ResponseWriter, r *http.Request, params GetV0QueueParams) {
	log.Ctx(r.Context()).Trace().Any("params", params).Msg("GetV0Queue")
	paginationParams := parseGetQueueParams(params)
	executions, err := a.getQueue(r.Context(), paginationParams)
	if err != nil {
		if errors.Is(err, ErrPageOutOfRange) {
			sendError(r.Context(), w, http.StatusBadRequest, "Page out of range", err.Error())
		} else if errors.Is(err, item.ErrSortNotSupported) {
			sendError(r.Context(), w, http.StatusBadRequest, "Sort parameter not supported", err.Error())
		} else {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not get executions", err.Error())
		}
		return
	}
	a.renderResponse(w, r, executions, "queue.html.tmpl")
}

func (a *amplifyAPI) renderResponse(w http.ResponseWriter, r *http.Request, data interface{}, templateFile string) {
	switch r.Header.Get("Accept") {
	case "application/json":
		fallthrough
	case "application/vnd.api+json":
		w.Header().Set("Content-Type", "application/vnd.api+json")
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render JSON", err.Error())
			return
		}
	default:
		err := a.writeHTML(w, templateFile, data)
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
	i, err := a.getItemDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendError(r.Context(), w, http.StatusNotFound, "Execution not found", fmt.Sprintf("Execution %s not found", id.String()))
		} else {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not get execution", err.Error())
		}
		return
	}
	queueDatum := &QueueDatum{
		Data: i,
	}
	a.renderResponse(w, r, queueDatum, "queueItem.html.tmpl")
}

// Run all workflows for a CID
// (PUT /v0/queue/{id})
func (a *amplifyAPI) PutV0QueueId(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	log.Ctx(r.Context()).Trace().Str("id", id.String()).Msg("PutV0QueueId")
	var body QueuePutDatum
	switch r.Header.Get("Content-Type") {
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
	if len(body.Data.Attributes.Inputs) == 0 {
		sendError(r.Context(), w, http.StatusBadRequest, "No inputs", "No inputs were provided.")
		return
	}
	a.createExecution(r.Context(), w, id, body.Data.Attributes.Inputs[0].Cid)
	a.GetV0QueueId(w, r, id)
}

// Run all workflows for a CID (not recommended)
// (POST /api/v0/queue)
func (a *amplifyAPI) PostV0Queue(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("PostV0Queue")
	var body ExecutionRequest
	switch r.Header.Get("Content-Type") {
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
	a.createExecution(r.Context(), w, uuid.New(), body.Cid)
}

func (a *amplifyAPI) createExecution(ctx context.Context, w http.ResponseWriter, executionID uuid.UUID, cid string) {
	err := a.er.Create(ctx, item.ItemParams{
		ID:       executionID,
		CID:      cid,
		Priority: true,
	})
	if err != nil {
		if err == queue.ErrQueueFull {
			sendError(ctx, w, http.StatusTooManyRequests, "Queue full", err.Error())
			return
		} else {
			sendError(ctx, w, http.StatusInternalServerError, "Could not create execution", err.Error())
			return
		}
	}
	w.WriteHeader(202)
}

func (a *amplifyAPI) CreateExecution(ctx context.Context, executionID uuid.UUID, cid string) error {
	return a.er.Create(ctx, item.ItemParams{
		ID:       executionID,
		CID:      cid,
		Priority: false,
	})
}

// Get Amplify work graph
// (GET /v0/graph)
func (a *amplifyAPI) GetV0Graph(w http.ResponseWriter, r *http.Request, params GetV0GraphParams) {
	log.Ctx(r.Context()).Trace().Msg("GetV0Graph")
	paginationParams := parsePaginationParams(params.PageSize, params.PageNumber)
	nn := a.tf.NodeNames()
	graph := make([]NodeSpec, len(nn))
	for idx, n := range nn {
		node, err := a.tf.GetNode(n)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not get node", err.Error())
			return
		}
		inputs := make([]NodeInput, len(node.Inputs))
		for i, v := range node.Inputs {
			inputs[i] = NodeInput{
				OutputId:  v.OutputID,
				Path:      v.Path,
				Root:      v.Root,
				NodeId:    v.NodeID,
				Predicate: v.Predicate,
			}
		}
		outputs := make([]NodeOutput, len(node.Outputs))
		for i, v := range node.Outputs {
			outputs[i] = NodeOutput{
				Id:   v.ID,
				Path: v.Path,
			}
		}
		graph[idx] = NodeSpec{
			Id: node.ID,
			Attributes: &NodeSpecAttributes{
				Inputs:  inputs,
				JobId:   node.JobID,
				Outputs: &outputs,
			},
			Links: &map[string]string{
				"self": fmt.Sprintf("/api/v0/graph/%s", node.ID),
			},
		}
	}
	totalPages := 1
	if paginationParams.PageNumber > totalPages || paginationParams.PageNumber < 1 {
		sendError(r.Context(), w, http.StatusBadRequest, "Page out of range", ErrPageOutOfRange.Error())
		return
	}
	outputGraph := &GraphCollection{
		Meta: &map[string]interface{}{
			"totalPages": totalPages,
			"count":      len(graph),
		},
		Data: graph,
		Links: &PaginationLinks{
			First: util.StrP(fmt.Sprintf("/api/v0/graph?page[size]=%d&page[number]=%d", paginationParams.PageSize, 1)),
			Last:  util.StrP(fmt.Sprintf("/api/v0/graph?page[size]=%d&page[number]=%d", paginationParams.PageSize, totalPages)),
			Next:  util.StrP(fmt.Sprintf("/api/v0/graph?page[size]=%d&page[number]=%d", paginationParams.PageSize, paginationParams.PageNumber+1)),
			Prev:  util.StrP(fmt.Sprintf("/api/v0/graph?page[size]=%d&page[number]=%d", paginationParams.PageSize, paginationParams.PageNumber-1)),
			AdditionalProperties: map[string]Link{
				"self": "/api/v0/graph",
				"home": "/api/v0",
			},
		},
	}
	a.renderResponse(w, r, outputGraph, "graph.html.tmpl")
}

// Job defines model for job.
type Job struct {
	Entrypoint *[]string `json:"entrypoint,omitempty"`
	Id         string    `json:"id"`
	Image      string    `json:"image"`
	Links      *Links    `json:"links,omitempty"`
	Type       string    `json:"type"`
}

func (a *amplifyAPI) getJob(jobId string) (*JobSpec, error) {
	j, err := a.tf.GetJob(jobId)
	if err != nil {
		return nil, err
	}
	return &JobSpec{
		Id:         j.ID,
		Type:       "JobSpec",
		Attributes: &JobSpecAttributes{Image: j.Image, Entrypoint: j.Entrypoint},
		Links: &map[string]string{
			"self": fmt.Sprintf("/api/v0/jobs/%s", j.ID),
			"list": "/api/v0/jobs",
			"home": "/api/v0",
		},
	}, nil
}

func (a *amplifyAPI) getQueue(ctx context.Context, params item.ListParams) (*QueueCollection, error) {
	log.Ctx(ctx).Trace().Msgf("GetQueue: %+v", params)
	e, err := a.er.List(ctx, params)
	if err != nil {
		return nil, err
	}
	list := make([]QueueItem, len(e))
	for i, item := range e {
		list[i] = *buildItem(ctx, item)
	}
	count, err := a.er.Count(ctx)
	if err != nil {
		return nil, err
	}
	totalPages := int(math.Ceil(float64(count) / float64(params.PageSize)))
	log.Ctx(ctx).Trace().Msgf("Total pages: %d", totalPages)
	if params.PageNumber > totalPages || params.PageNumber < 1 {
		return nil, ErrPageOutOfRange
	}

	return &QueueCollection{
		Data: list,
		Meta: &map[string]interface{}{
			"totalPages": &totalPages,
			"count":      &count,
		},
		Links: &PaginationLinks{
			First: util.StrP(fmt.Sprintf("/api/v0/queue?page[size]=%d&page[number]=%d", params.PageSize, 1)),
			Last:  util.StrP(fmt.Sprintf("/api/v0/queue?page[size]=%d&page[number]=%d", params.PageSize, totalPages)),
			Next:  util.StrP(fmt.Sprintf("/api/v0/queue?page[size]=%d&page[number]=%d", params.PageSize, params.PageNumber+1)),
			Prev:  util.StrP(fmt.Sprintf("/api/v0/queue?page[size]=%d&page[number]=%d", params.PageSize, params.PageNumber-1)),
			AdditionalProperties: map[string]Link{
				"self": "/api/v0/queue",
				"home": "/api/v0",
			},
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
		Id:   child.Name,
		Type: "Node",
		Attributes: &NodeAttributes{
			Inputs:  inputs,
			Outputs: &outputs,
			Result: &ItemResult{
				Id:      util.StrP(child.Results.ID),
				Stderr:  util.StrP(child.Results.StdErr),
				Stdout:  util.StrP(child.Results.StdOut),
				Skipped: util.BoolP(child.Results.Skipped),
			},
		},
		Links: &map[string]string{
			"self": fmt.Sprintf("/api/v0/queue/%s", rootId),
			"list": "/api/v0/queue",
		},
		Meta: &QueueMetadata{
			Submitted: child.Metadata.CreatedAt.Format(time.RFC3339),
			Status:    child.Metadata.Status,
			Started:   util.StrP(child.Metadata.StartedAt.Format(time.RFC3339)),
			Ended:     util.StrP(child.Metadata.EndedAt.Format(time.RFC3339)),
		},
	}
	if len(child.Children) > 0 {
		children := make([]Node, len(child.Children))
		for idx, c := range child.Children {
			children[idx] = makeNode(ctx, c, rootId)
		}
		node.Attributes.Children = &children
	}
	return node
}

func (a *amplifyAPI) getItemDetail(ctx context.Context, executionId openapi_types.UUID) (*QueueItemDetail, error) {
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

	v := &QueueItemDetail{
		Id:   i.ID.String(),
		Type: "QueueItemDetail",
		Attributes: &QueueItemAttributes{
			Inputs: []ExecutionRequest{{
				Cid: i.CID,
			}},
			Outputs: &outputs,
			Graph:   &dag,
		},
		Meta: &QueueMetadata{
			Submitted: i.Metadata.CreatedAt.Format(time.RFC3339),
			Started:   util.StrP(i.Metadata.StartedAt.Format(time.RFC3339)),
			Ended:     util.StrP(i.Metadata.EndedAt.Format(time.RFC3339)),
			Status:    childStatusesToList(ctx, i.RootNodes),
		},
		Links: &map[string]string{
			"self": fmt.Sprintf("/api/v0/queue/%s", i.ID),
			"list": "/api/v0/queue",
		},
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

func buildItem(ctx context.Context, i *item.Item) *QueueItem {
	v := QueueItem{
		Id:   i.ID.String(),
		Type: "QueueItem",
		Meta: &QueueMetadata{
			Submitted: i.Metadata.CreatedAt.Format(time.RFC3339),
			Started:   util.StrP(i.Metadata.StartedAt.Format(time.RFC3339)),
			Ended:     util.StrP(i.Metadata.EndedAt.Format(time.RFC3339)),
			Status:    childStatusesToList(ctx, i.RootNodes),
		},
		Links: &map[string]string{
			"self": fmt.Sprintf("/api/v0/queue/%s", i.ID),
			"list": "/api/v0/queue",
		},
	}
	if !i.Metadata.StartedAt.IsZero() {
		v.Meta.Started = util.StrP(i.Metadata.StartedAt.Format(time.RFC3339))
	}
	if !i.Metadata.EndedAt.IsZero() {
		v.Meta.Ended = util.StrP(i.Metadata.EndedAt.Format(time.RFC3339))
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
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(&Error{
		Title:  &userErr,
		Detail: &devErr,
		Status: util.StrP(fmt.Sprintf("%d", statusCode)),
		Links: &map[string]string{
			"home": "/api/v0",
		},
	})
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
