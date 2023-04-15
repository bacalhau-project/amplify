// Package api provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.12.4 DO NOT EDIT.
package api

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/deepmap/oapi-codegen/pkg/runtime"
	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gorilla/mux"
)

// Error defines model for error.
type Error struct {
	// Detail A human-readable explanation specific to this occurrence of the problem.
	Detail *string `json:"detail,omitempty"`

	// Title A short, human-readable summary of the problem that SHOULD NOT change from occurrence to occurrence of the problem, except for purposes of localization.
	Title *string `json:"title,omitempty"`
}

// Errors defines model for errors.
type Errors = []Error

// ExecutionRequest defines model for executionRequest.
type ExecutionRequest struct {
	Cid string `json:"cid"`
}

// Graph defines model for graph.
type Graph struct {
	Data  *[]NodeConfig `json:"data,omitempty"`
	Links *Links        `json:"links,omitempty"`
}

// Home defines model for home.
type Home struct {
	Links *Links  `json:"links,omitempty"`
	Type  *string `json:"type,omitempty"`
}

// Item defines model for item.
type Item struct {
	Id       string       `json:"id"`
	Links    *Links       `json:"links,omitempty"`
	Metadata ItemMetadata `json:"metadata"`
	Type     string       `json:"type"`
}

// ItemMetadata defines model for itemMetadata.
type ItemMetadata struct {
	Ended     *string `json:"ended,omitempty"`
	Started   *string `json:"started,omitempty"`
	Status    string  `json:"status"`
	Submitted string  `json:"submitted"`
}

// ItemResult defines model for itemResult.
type ItemResult struct {
	// Id External execution ID
	Id *string `json:"id,omitempty"`

	// Skipped Whether this node was skipped due to predicates not matching.
	Skipped *bool   `json:"skipped,omitempty"`
	Stderr  *string `json:"stderr,omitempty"`
	Stdout  *string `json:"stdout,omitempty"`
}

// Job defines model for job.
type Job struct {
	Entrypoint *[]string `json:"entrypoint,omitempty"`
	Id         string    `json:"id"`
	Image      string    `json:"image"`
	Links      *Links    `json:"links,omitempty"`
	Type       string    `json:"type"`
}

// Jobs defines model for jobs.
type Jobs struct {
	Data  *[]Job `json:"data,omitempty"`
	Links *Links `json:"links,omitempty"`
}

// Links defines model for links.
type Links = map[string]interface{}

// Node defines model for node.
type Node struct {
	Children *[]Node            `json:"children,omitempty"`
	Id       openapi_types.UUID `json:"id"`
	Inputs   []ExecutionRequest `json:"inputs"`
	Links    *Links             `json:"links,omitempty"`
	Metadata ItemMetadata       `json:"metadata"`
	Name     *string            `json:"name,omitempty"`
	Outputs  []ExecutionRequest `json:"outputs"`
	Result   *ItemResult        `json:"result,omitempty"`
	Type     string             `json:"type"`
}

// NodeConfig Static configuration of a node.
type NodeConfig struct {
	Id      *string       `json:"id,omitempty"`
	Inputs  *[]NodeInput  `json:"inputs,omitempty"`
	JobId   *string       `json:"job_id,omitempty"`
	Outputs *[]NodeOutput `json:"outputs,omitempty"`
}

// NodeInput Input specification for a node.
type NodeInput struct {
	OutputId  *string `json:"output_id,omitempty"`
	Path      *string `json:"path,omitempty"`
	Predicate *string `json:"predicate,omitempty"`
	Root      *bool   `json:"root,omitempty"`
	StepId    *string `json:"step_id,omitempty"`
}

// NodeOutput Output specification for a node.
type NodeOutput struct {
	Id   *string `json:"id,omitempty"`
	Path *string `json:"path,omitempty"`
}

// PageMeta defines model for pageMeta.
type PageMeta struct {
	// TotalPages Total number of pages in paginated result.
	TotalPages *int `json:"totalPages,omitempty"`
}

// Queue defines model for queue.
type Queue struct {
	Data  *[]Item   `json:"data,omitempty"`
	Links *Links    `json:"links,omitempty"`
	Meta  *PageMeta `json:"meta,omitempty"`
}

// PostApiV0QueueFormdataRequestBody defines body for PostApiV0Queue for application/x-www-form-urlencoded ContentType.
type PostApiV0QueueFormdataRequestBody = ExecutionRequest

// PutApiV0QueueIdJSONRequestBody defines body for PutApiV0QueueId for application/json ContentType.
type PutApiV0QueueIdJSONRequestBody = ExecutionRequest

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Amplify Home
	// (GET /)
	Get(w http.ResponseWriter, r *http.Request)
	// Amplify V0 API Home
	// (GET /api/v0)
	GetApiV0(w http.ResponseWriter, r *http.Request)
	// Get Amplify work graph
	// (GET /api/v0/graph)
	GetApiV0Graph(w http.ResponseWriter, r *http.Request)
	// List all Amplify jobs
	// (GET /api/v0/jobs)
	GetApiV0Jobs(w http.ResponseWriter, r *http.Request)
	// Get a job by id
	// (GET /api/v0/jobs/{id})
	GetApiV0JobsId(w http.ResponseWriter, r *http.Request, id string)
	// Amplify work queue
	// (GET /api/v0/queue)
	GetApiV0Queue(w http.ResponseWriter, r *http.Request)
	// Run all workflows for a CID (not recommended)
	// (POST /api/v0/queue)
	PostApiV0Queue(w http.ResponseWriter, r *http.Request)
	// Get an item from the queue by id
	// (GET /api/v0/queue/{id})
	GetApiV0QueueId(w http.ResponseWriter, r *http.Request, id openapi_types.UUID)
	// Run all workflows for a CID
	// (PUT /api/v0/queue/{id})
	PutApiV0QueueId(w http.ResponseWriter, r *http.Request, id openapi_types.UUID)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandlerFunc   func(w http.ResponseWriter, r *http.Request, err error)
}

type MiddlewareFunc func(http.HandlerFunc) http.HandlerFunc

// Get operation middleware
func (siw *ServerInterfaceWrapper) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.Get(w, r)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// GetApiV0 operation middleware
func (siw *ServerInterfaceWrapper) GetApiV0(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetApiV0(w, r)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// GetApiV0Graph operation middleware
func (siw *ServerInterfaceWrapper) GetApiV0Graph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetApiV0Graph(w, r)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// GetApiV0Jobs operation middleware
func (siw *ServerInterfaceWrapper) GetApiV0Jobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetApiV0Jobs(w, r)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// GetApiV0JobsId operation middleware
func (siw *ServerInterfaceWrapper) GetApiV0JobsId(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// ------------- Path parameter "id" -------------
	var id string

	err = runtime.BindStyledParameter("simple", false, "id", mux.Vars(r)["id"], &id)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "id", Err: err})
		return
	}

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetApiV0JobsId(w, r, id)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// GetApiV0Queue operation middleware
func (siw *ServerInterfaceWrapper) GetApiV0Queue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetApiV0Queue(w, r)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// PostApiV0Queue operation middleware
func (siw *ServerInterfaceWrapper) PostApiV0Queue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.PostApiV0Queue(w, r)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// GetApiV0QueueId operation middleware
func (siw *ServerInterfaceWrapper) GetApiV0QueueId(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// ------------- Path parameter "id" -------------
	var id openapi_types.UUID

	err = runtime.BindStyledParameter("simple", false, "id", mux.Vars(r)["id"], &id)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "id", Err: err})
		return
	}

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetApiV0QueueId(w, r, id)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// PutApiV0QueueId operation middleware
func (siw *ServerInterfaceWrapper) PutApiV0QueueId(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// ------------- Path parameter "id" -------------
	var id openapi_types.UUID

	err = runtime.BindStyledParameter("simple", false, "id", mux.Vars(r)["id"], &id)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "id", Err: err})
		return
	}

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.PutApiV0QueueId(w, r, id)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

type UnescapedCookieParamError struct {
	ParamName string
	Err       error
}

func (e *UnescapedCookieParamError) Error() string {
	return fmt.Sprintf("error unescaping cookie parameter '%s'", e.ParamName)
}

func (e *UnescapedCookieParamError) Unwrap() error {
	return e.Err
}

type UnmarshallingParamError struct {
	ParamName string
	Err       error
}

func (e *UnmarshallingParamError) Error() string {
	return fmt.Sprintf("Error unmarshalling parameter %s as JSON: %s", e.ParamName, e.Err.Error())
}

func (e *UnmarshallingParamError) Unwrap() error {
	return e.Err
}

type RequiredParamError struct {
	ParamName string
}

func (e *RequiredParamError) Error() string {
	return fmt.Sprintf("Query argument %s is required, but not found", e.ParamName)
}

type RequiredHeaderError struct {
	ParamName string
	Err       error
}

func (e *RequiredHeaderError) Error() string {
	return fmt.Sprintf("Header parameter %s is required, but not found", e.ParamName)
}

func (e *RequiredHeaderError) Unwrap() error {
	return e.Err
}

type InvalidParamFormatError struct {
	ParamName string
	Err       error
}

func (e *InvalidParamFormatError) Error() string {
	return fmt.Sprintf("Invalid format for parameter %s: %s", e.ParamName, e.Err.Error())
}

func (e *InvalidParamFormatError) Unwrap() error {
	return e.Err
}

type TooManyValuesForParamError struct {
	ParamName string
	Count     int
}

func (e *TooManyValuesForParamError) Error() string {
	return fmt.Sprintf("Expected one value for %s, got %d", e.ParamName, e.Count)
}

// Handler creates http.Handler with routing matching OpenAPI spec.
func Handler(si ServerInterface) http.Handler {
	return HandlerWithOptions(si, GorillaServerOptions{})
}

type GorillaServerOptions struct {
	BaseURL          string
	BaseRouter       *mux.Router
	Middlewares      []MiddlewareFunc
	ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)
}

// HandlerFromMux creates http.Handler with routing matching OpenAPI spec based on the provided mux.
func HandlerFromMux(si ServerInterface, r *mux.Router) http.Handler {
	return HandlerWithOptions(si, GorillaServerOptions{
		BaseRouter: r,
	})
}

func HandlerFromMuxWithBaseURL(si ServerInterface, r *mux.Router, baseURL string) http.Handler {
	return HandlerWithOptions(si, GorillaServerOptions{
		BaseURL:    baseURL,
		BaseRouter: r,
	})
}

// HandlerWithOptions creates http.Handler with additional options
func HandlerWithOptions(si ServerInterface, options GorillaServerOptions) http.Handler {
	r := options.BaseRouter

	if r == nil {
		r = mux.NewRouter()
	}
	if options.ErrorHandlerFunc == nil {
		options.ErrorHandlerFunc = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
		ErrorHandlerFunc:   options.ErrorHandlerFunc,
	}

	r.HandleFunc(options.BaseURL+"/", wrapper.Get).Methods("GET")

	r.HandleFunc(options.BaseURL+"/api/v0", wrapper.GetApiV0).Methods("GET")

	r.HandleFunc(options.BaseURL+"/api/v0/graph", wrapper.GetApiV0Graph).Methods("GET")

	r.HandleFunc(options.BaseURL+"/api/v0/jobs", wrapper.GetApiV0Jobs).Methods("GET")

	r.HandleFunc(options.BaseURL+"/api/v0/jobs/{id}", wrapper.GetApiV0JobsId).Methods("GET")

	r.HandleFunc(options.BaseURL+"/api/v0/queue", wrapper.GetApiV0Queue).Methods("GET")

	r.HandleFunc(options.BaseURL+"/api/v0/queue", wrapper.PostApiV0Queue).Methods("POST")

	r.HandleFunc(options.BaseURL+"/api/v0/queue/{id}", wrapper.GetApiV0QueueId).Methods("GET")

	r.HandleFunc(options.BaseURL+"/api/v0/queue/{id}", wrapper.PutApiV0QueueId).Methods("PUT")

	return r
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/+xZbXPbuBH+Kxi0H9qpXmjZPiX65iaZnK93lzTxXWeaeDoguRThgACMF9tKRv+9A4Ci",
	"RBKypDhJ4+l9sSkCWOw+u/ssFvyEM1FJwYEbjWefsM5KqIh/BKWEcg9SCQnKUPCvczCEsvCkM0WloYLj",
	"GT5Dpa0IHyogOUkZILiTjHDihpGWkNGCZsgIZEqqkcgyqxTwDJAokCkBSSVSBtUIDzDckUoywDP8TFiW",
	"Iy4MKijPEUEKCgjLjEAEXYkUZYQxyNF7XIEhOTHkPcYDbBbSCdBGUT7HywE21DiRfbV1KZQZdLXXtqqI",
	"WnS0Q6YkBr398dVvPz9Hv766QFlJ+BxQoUS1aZMR2y0cILjLQBpUCIWkVVJo0G4OExlh9KNHrA3DTyIN",
	"IAjL875xy+aNSK8gM85c7z3vMGqg8g9/VlDgGf7TeO3yce3vcXD2WhBRiiy8nDvIrNPoDVxb0KYfEBnN",
	"fbg02qakWKRA5/lCfTSnupBTm1fT0k5/sOV0MfmBF8dQ2AW7JmlxLDI2N9eL09Mi/ZjTqG0Kri1VkOPZ",
	"O7/ZZcTcuSKyjAQrMWRvDLjI4ZngBZ3HgGCUf9gpIkyKOqQUFfQVPETqSugm1l7qXgHhIOjv3/Xd9CQh",
	"p0+mT4cnk6fZ8ISQyfDpk2Q6hHSSJOkRgenpSSy7DrNjlam7Fjidf1nNjdrvrdoVNLTJmY2tL7dg9MuG",
	"bm2sgOfQgWuSHD0dHiXDo8lFMp1NktlpMjqd/DuGkDZEmYetN1a3lyvLuRuOTbdpRc3nb9iBcC2u0WQb",
	"gG9AW2a2hVqbfV/cGVCcMNTQDDp/3qK+PQJyErX/A5USIlv+qwRTggplyKU8uiUa1dNRbj15SwU5zYgB",
	"7Xm3IiYrKZ+3WLkgTEOzcyoEA8KDp3JQqo27341qpEUFqJ4QdXIurLl/qZuwV8ZfiTQWxEYtpKC8vcs7",
	"zDQe4OEb59aGLftFtEOKXfpositiG63IvJO9NrXc2NnRk1HyBVilzw4Ogb3IIehWT72MY6kfWF6cMl+h",
	"rjTreyMuuCPluqQsV8Db7t98/y7k6n6pt7wc7D37CLepvybUe0ipIc375qyZbuus5XIzrHcdArZHeiFU",
	"RYyLXEvzaJBzac0Bx67u6eqBAfL5pZWTCuK5PPSQRGwV1nxxY1VTPHYpX5eZaN7HNY4l/gZh1a5b23UP",
	"HWwcFXsF5q0hhmYo88NWhQZIFIj4auNKSKwuPjSSnOxztySG6pVI/7Nlm0Od6PZ55df0N1puQSro1QPK",
	"v25aw4CT64m2ARVU3WaIJKaMD6xqeXRUCWE2BlplHGR8s2121rj0DA3vD7D0MBNj6kgyB5feff43whD2",
	"msxX3fymohduDHFbpaBcyDopGlHuHignBnIUErR1EDo6bhSg3MAcVFynawsWHlhE/Xn/i5DkrskNghFb",
	"lj4/CxG5Tqgko8UCSZsymqGz1+e4uXpYDeIBvgGlw4JklIyOfCJK4ERSPMPHo2R0jIO7vU1j92cOPrIc",
	"cj6Azl29ewnuLKhAS8F1gHOSJL7MC24gHPIM3JlxaSq2vuBp0+V7myTHmZvhnyD8TkW+CL//LvIFclLC",
	"wHg9Ur9YL42w7nLQTYd/eEzrG5YN0H50zawbGhNJxzfJfWafSfp7stt2IiWrM258w/MRkfRvV1rwNhT3",
	"xUG5UurRwPh74uKuh+a4uSOpMW2L+5lqgwhjnpJ80psS0Ermuj8LUgZbPPKyHv3qbglqfJd++TDqOOYl",
	"mAbIW6E+oLXyK9+s+ouoa844IvXyK5G6XpAgTfmcuX4QpPMV8YILJm5H6MI1jMBz3+UhRrXR3rFuj9FW",
	"z/3kNPgGjvOWPg6/NSlxtkZf99w2/kTz5U6qcvCe557VFanAgNK+02rr8Cupmuti52oj0NxTPHWjvv6v",
	"jurh/Lo+0BplYbCBY9fey2/j2+/XtQN8kpx8cZPre/bv0er2V4M+JYXPJ+kC0bwV1M05bWudCHyyYhyN",
	"CM99bqyqRpCwjWn+WY9+9XAMajwOrmnVh1rxAZZCR1zQJniqfS9RCFUhbaUUyiDB2WKAqB90IaAgE1Xl",
	"71v8ZC5URRiyGlAKGXH/15NplYMUzg0j9BYAvf7tArViIzBe172vhe7617f5Drp7XHs3vL29HTrth1Yx",
	"4Jmob9n3zL/epUK7z3e0uOzF2iRSZLMMpIG8Zork/4kpXjjVUH2RRvkc1a7rBukbyzuZH7rYZ+fP0V86",
	"cfbXPqfsVyl9+MRK5eE1cMd13TepieFC8dGcUwlHrscOX7QbLl9ViQGOXnC84H6W9h/ofeS4k4uKRouT",
	"Oac3wF3U9EnE/m+CYB+mOjDd/yCm74KYglgN6mYVQlYxPMNjvLxc/jcAAP//vhdAA/8iAAA=",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %s", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	var res = make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	var resolvePath = PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		var pathToFile = url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
