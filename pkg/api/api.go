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

	"github.com/bacalhau-project/amplify/pkg/job"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/bacalhau-project/amplify/pkg/util"
	"github.com/bacalhau-project/amplify/pkg/workflow"
	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
	"github.com/rs/zerolog/log"
)

var _ ServerInterface = (*amplifyAPI)(nil)

type amplifyAPI struct {
	*sync.Mutex
	jf   *job.JobFactory
	wf   *workflow.WorkflowFactory
	er   queue.QueueRepository
	tf   *task.TaskFactory
	tmpl *template.Template
}

// TODO: Getting gross
func NewAmplifyAPI(jf *job.JobFactory, wf *workflow.WorkflowFactory, er queue.QueueRepository, tf *task.TaskFactory) *amplifyAPI {
	return &amplifyAPI{
		jf:   jf,
		wf:   wf,
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
		if errors.Is(err, job.ErrJobNotFound) {
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
	e, err := a.getItem(r.Context(), id)
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

// Enqueue a task
// (PUT /v0/queue/workflow/{id})
func (a *amplifyAPI) PutV0QueueWorkflowId(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	log.Ctx(r.Context()).Trace().Str("id", id.String()).Msg("PutV0QueueWorkflowId")
	// Parse request body
	var body ExecutionRequest
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		sendError(r.Context(), w, http.StatusBadRequest, "Could not parse request body", err.Error())
		return
	}
	task, err := a.tf.CreateWorkflowTask(r.Context(), *body.Name, *body.Cid)
	if err != nil {
		sendError(r.Context(), w, http.StatusInternalServerError, "Could not create task", err.Error())
		return
	}
	err = a.er.Create(r.Context(), queue.CreateRequest{
		ID:   id.String(),
		Kind: "workflow",
		Name: *body.Name,
		CID:  *body.Cid,
		Task: task,
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
		if errors.Is(err, workflow.ErrWorkflowNotFound) {
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
	jobList := make([]Job, len(a.jf.JobNames()))
	for i, id := range a.jf.JobNames() {
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
	j, err := a.jf.GetJob(jobId)
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
	wfList := make([]Workflow, len(a.wf.WorkflowNames()))
	for i, id := range a.wf.WorkflowNames() {
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
	w, err := a.wf.GetWorkflow(workflowId)
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
		return e[i].Submitted.UnixNano() < e[j].Submitted.UnixNano()
	})
	executions := make([]Item, len(e))
	for i, execution := range e {
		executions[i] = *buildItem(execution)
	}
	return &Queue{
		Data: &executions,
		Links: &Links{
			"self": "/api/v0/queue",
			"home": "/api/v0",
		},
	}, nil
}

func (a *amplifyAPI) getItem(ctx context.Context, executionId openapi_types.UUID) (*Item, error) {
	e, err := a.er.Get(ctx, executionId.String())
	if err != nil {
		return nil, err
	}
	return buildItem(e), nil
}

func buildItem(i queue.Item) *Item {
	v := Item{
		Id:        util.StrP(i.ID),
		Type:      util.StrP("item"),
		Kind:      util.StrP(i.Kind),
		Name:      util.StrP(i.Name),
		Cid:       util.StrP(i.CID),
		Submitted: util.StrP(i.Submitted.Format(time.RFC3339)),
		Links: &Links{
			"self": fmt.Sprintf("/api/v0/queue/%s", i.ID),
			"list": "/api/v0/queue",
		},
	}
	if !i.Started.IsZero() {
		v.Started = util.StrP(i.Started.Format(time.RFC3339))
	}
	if !i.Ended.IsZero() {
		v.Ended = util.StrP(i.Ended.Format(time.RFC3339))
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
		buf.WriteTo(w)
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
	json.NewEncoder(w).Encode(&Errors{e})
}
