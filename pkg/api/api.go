package api

import (
	"bytes"
	"context"
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
	"github.com/rs/zerolog/log"
)

var _ ServerInterface = (*amplifyAPI)(nil)

type amplifyAPI struct {
	*sync.Mutex
	er   queue.QueueRepository
	tf   *task.TaskFactory
	tmpl *template.Template
}

// TODO: Getting gross
func NewAmplifyAPI(er queue.QueueRepository, tf *task.TaskFactory) *amplifyAPI {
	return &amplifyAPI{
		er:   er,
		tf:   tf,
		tmpl: template.Must(template.ParseGlob("pkg/api/templates/*.tmpl")),
	}
}

// Amplify home
// (GET /v0)
func (a *amplifyAPI) GetV0(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("GetV0")
	err := a.writeHTML(w, "home.html.tmpl", &Home{
		Type: util.StrP("home"),
		Links: util.MapP(map[string]interface{}{
			"self":      "/api/v0/",
			"queue":     "/api/v0/queue",
			"jobs":      "/api/v0/jobs",
			"workflows": "/api/v0/workflows",
		}),
	})
	if err != nil {
		sendError(r.Context(), w, http.StatusInternalServerError, "Could not render HTML", err.Error())
		return
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
	allWorkflows := a.tf.WorkflowNames()
	var workflows []task.Workflow
	for _, wf := range allWorkflows {
		twf, err := a.tf.GetWorkflow(wf)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not get workflow", err.Error())
			return
		}
		workflows = append(workflows, twf)
	}
	task, err := a.tf.CreateTask(r.Context(), workflows, body.Cid)
	if err != nil {
		sendError(r.Context(), w, http.StatusInternalServerError, "Could not create task", err.Error())
		return
	}
	err = a.er.Create(r.Context(), queue.Item{
		ID:  id.String(),
		CID: body.Cid,
		Dag: task,
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

// Enqueue a task
// (PUT /v0/queue/workflow/{id})
func (a *amplifyAPI) PutV0QueueWorkflowId(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	log.Ctx(r.Context()).Trace().Str("id", id.String()).Msg("PutV0QueueWorkflowId")
	// Parse request body
	var body WorkflowExecutionRequest
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		sendError(r.Context(), w, http.StatusBadRequest, "Could not parse request body", err.Error())
		return
	}
	wf, err := a.tf.GetWorkflow(body.Name)
	if err != nil {
		sendError(r.Context(), w, http.StatusBadRequest, "Could not get workflow", err.Error())
		return
	}
	task, err := a.tf.CreateTask(r.Context(), []task.Workflow{wf}, body.Cid)
	if err != nil {
		sendError(r.Context(), w, http.StatusInternalServerError, "Could not create task", err.Error())
		return
	}
	err = a.er.Create(r.Context(), queue.Item{
		ID:  id.String(),
		CID: body.Cid,
		Dag: task,
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

// List all Amplify workflows
// (GET /v0/workflows)
func (a *amplifyAPI) GetV0Workflows(w http.ResponseWriter, r *http.Request) {
	log.Ctx(r.Context()).Trace().Msg("GetV0Workflows")
	workflows, err := a.getWorkflows()
	if err != nil {
		sendError(r.Context(), w, http.StatusInternalServerError, "Could not get workflows", err.Error())
		return
	}
	switch r.Header.Get("Content-type") {
	case "application/json":
		fallthrough
	case "application/vnd.api+json":
		w.Header().Set("Content-Type", "application/vnd.api+json")
		err := json.NewEncoder(w).Encode(workflows)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render JSON", err.Error())
			return
		}
	default:
		err := a.writeHTML(w, "workflows.html.tmpl", workflows)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render HTML", err.Error())
			return
		}
	}
}

// Get a workflow by id
// (GET /v0/workflows/{id})
func (a *amplifyAPI) GetV0WorkflowsId(w http.ResponseWriter, r *http.Request, id string) {
	log.Ctx(r.Context()).Trace().Str("id", id).Msg("GetV0WorkflowsId")
	wf, err := a.getWorkflow(id)
	if err != nil {
		if errors.Is(err, task.ErrWorkflowNotFound) {
			sendError(r.Context(), w, http.StatusNotFound, "Workflow not found", fmt.Sprintf("Workflow %s not found", id))
		} else {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not get workflow", err.Error())
		}
		return
	}
	switch r.Header.Get("Content-type") {
	case "application/json":
		fallthrough
	case "application/vnd.api+json":
		w.Header().Set("Content-Type", "application/vnd.api+json")
		err := json.NewEncoder(w).Encode(wf)
		if err != nil {
			sendError(r.Context(), w, http.StatusInternalServerError, "Could not render JSON", err.Error())
			return
		}
	default:
		err := a.writeHTML(w, "workflow.html.tmpl", wf)
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
		},
	}, nil
}

func (a *amplifyAPI) getJob(jobId string) (*Job, error) {
	j, err := a.tf.GetJob(jobId)
	if err != nil {
		return nil, err
	}
	return &Job{
		Type: util.StrP("job"),
		Id:   util.StrP(j.Name),
		Links: &Links{
			"self": fmt.Sprintf("/api/v0/jobs/%s", j.Name),
			"list": "/api/v0/jobs",
		},
	}, nil
}

func (a *amplifyAPI) getWorkflows() (*Workflows, error) {
	wfList := make([]Workflow, len(a.tf.WorkflowNames()))
	for i, id := range a.tf.WorkflowNames() {
		w, err := a.getWorkflow(id)
		if err != nil {
			return nil, err
		}
		wfList[i] = *w
	}
	return &Workflows{
		Data: &wfList,
		Links: &Links{
			"self": "/api/v0/workflows",
		},
	}, nil
}

func (a *amplifyAPI) getWorkflow(workflowId string) (*Workflow, error) {
	w, err := a.tf.GetWorkflow(workflowId)
	if err != nil {
		return nil, err
	}
	workflowJobs := make([]Job, len(w.Jobs))
	for i, job := range w.Jobs {
		j, err := a.getJob(job.Name)
		if err != nil {
			return nil, err
		}
		workflowJobs[i] = *j
	}
	return &Workflow{
		Type: util.StrP("workflow"),
		Id:   util.StrP(w.Name),
		Jobs: &workflowJobs,
		Links: &Links{
			"self": fmt.Sprintf("/api/v0/workflows/%s", w.Name),
			"list": "/api/v0/workflows",
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

func makeNode(child *dag.Node[[]string], rootId openapi_types.UUID) Node {
	inputs := make([]ExecutionRequest, len(child.Input()))
	for idx, input := range child.Input() {
		inputs[idx] = ExecutionRequest{
			Cid: input,
		}
	}
	node := Node{
		Id:     rootId,
		Inputs: inputs,
		Links: &Links{
			"self": fmt.Sprintf("/api/v0/queue/%s", rootId),
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

func (a *amplifyAPI) getItemDetail(ctx context.Context, executionId openapi_types.UUID) (*ItemDetail, error) {
	i, err := a.er.Get(ctx, executionId.String())
	if err != nil {
		return nil, err
	}
	dag := make([]Node, len(i.Dag))
	for idx, child := range i.Dag {
		dag[idx] = makeNode(child, executionId)
	}
	v := &ItemDetail{
		Id:   util.StrP(i.ID),
		Type: util.StrP("itemDetail"),
		Request: &ExecutionRequest{
			Cid: i.CID,
		},
		Metadata: &ItemMetadata{
			Submitted: i.Metadata.CreatedAt.Format(time.RFC3339),
		},
		Links: &Links{
			"self": fmt.Sprintf("/api/v0/queue/%s", i.ID),
			"list": "/api/v0/queue",
		},
		Dag: &dag,
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
		Id:   util.StrP(i.ID),
		Type: util.StrP("item"),
		Request: &ExecutionRequest{
			Cid: i.CID,
		},
		Metadata: &ItemMetadata{
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
