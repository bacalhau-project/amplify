package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bacalhau-project/amplify/pkg/api"
	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/config"
	"github.com/bacalhau-project/amplify/pkg/job"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/bacalhau-project/amplify/pkg/workflow"
	middleware "github.com/deepmap/oapi-codegen/pkg/chi-middleware"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const baseURL = "/api"

func newServeCommand(appContext cli.AppContext) *cobra.Command {
	return &cobra.Command{
		Use:     "serve",
		Short:   "Start the Amplify daemon",
		Long:    "The serve command starts the Amplify daemon and serves the REST API",
		Example: `amplify serve`,
		RunE:    executeServeCommand(appContext),
	}
}

func executeServeCommand(appContext cli.AppContext) runEFunc {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		swagger, err := api.GetSwagger()
		if err != nil {
			log.Ctx(ctx).Fatal().Err(err).Msg("Failed to load swagger spec")
		}

		// Clear out the servers array in the swagger spec, that skips validating
		// that server names match. We don't know how this thing will be run.
		swagger.Servers = openapi3.Servers{
			&openapi3.Server{
				URL: baseURL,
			},
		}

		// Config
		conf, err := config.GetConfig(appContext.Config.ConfigPath)
		if err != nil {
			return err
		}

		// Job Factory
		jobFactory := job.NewJobFactory(*conf)

		// Workflow Factory
		workflowFactory := workflow.NewWorkflowFactory(*conf)

		// Queue
		numWorkers := 10
		q := queue.NewInMemoryQueue(ctx, numWorkers)
		q.Start()
		queueRepository := queue.NewQueueRepository(q)

		// Task Factory
		taskFactory, err := task.NewTaskFactory(appContext)
		if err != nil {
			return err
		}

		store := api.NewAmplifyAPI(&jobFactory, &workflowFactory, queueRepository, taskFactory)

		// This is how you set up a basic Gorilla router
		r := mux.NewRouter()

		// Use our validation middleware to check all requests against the
		// OpenAPI schema.
		r.Use(middleware.OapiRequestValidator(swagger))

		api.HandlerFromMuxWithBaseURL(store, r, baseURL)

		s := &http.Server{
			Handler: r,
			Addr:    fmt.Sprintf("0.0.0.0:%d", 8081),
		}

		go func() {
			if err = s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Ctx(ctx).Fatal().Err(err).Msg("Failed to start HTTP server")
			}
		}()
		<-ctx.Done()
		log.Ctx(ctx).Info().Msg("Shutting down HTTP server")

		ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer func() {
			cancel()
		}()

		if err = s.Shutdown(ctxShutDown); err != nil {
			log.Ctx(ctx).Err(err).Msgf("server Shutdown Failed:%+s", err)
		}

		log.Ctx(ctx).Info().Msg("HTTP server shut down")
		if err == http.ErrServerClosed {
			err = nil
		}
		return nil
	}
}
