package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bacalhau-project/amplify/pkg/api"
	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/bacalhau-project/amplify/pkg/triggers"
	middleware "github.com/deepmap/oapi-codegen/pkg/chi-middleware"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const baseURL = "/api"

func newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "serve",
		Short:   "Start the Amplify daemon",
		Long:    "The serve command starts the Amplify daemon and serves the REST API",
		Example: `amplify serve`,
		RunE: func(cmd *cobra.Command, args []string) error {
			appContext := cli.DefaultAppContext(cmd)
			return executeServeCommand(appContext)(cmd, args)
		},
	}
}

func executeServeCommand(appContext cli.AppContext) runEFunc {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		swagger, err := api.GetSwagger()
		if err != nil {
			log.Ctx(ctx).Fatal().Err(err).Msg("Failed to load swagger spec")
			return err
		}

		// Clear out the servers array in the swagger spec, that skips validating
		// that server names match. We don't know how this thing will be run.
		swagger.Servers = openapi3.Servers{
			&openapi3.Server{
				URL: baseURL,
			},
		}

		// DAG Queue
		dagQueue, err := queue.NewGenericQueue(ctx, 10, 1024)
		if err != nil {
			return err
		}
		dagQueue.Start()
		defer dagQueue.Stop()
		queueRepository := queue.NewQueueRepository(dagQueue)

		// Job Queue
		jobQueue, err := queue.NewGenericQueue(ctx, 10, 1024)
		if err != nil {
			return err
		}
		jobQueue.Start()
		defer jobQueue.Stop()

		// Task Factory
		taskFactory, err := task.NewTaskFactory(appContext, jobQueue)
		if err != nil {
			return err
		}

		store := api.NewAmplifyAPI(queueRepository, taskFactory)

		// Setup the Triggers
		cidChan := make(chan cid.Cid)
		if appContext.Config.Trigger.IPFSSearch.Enabled {
			t := triggers.IPFSSearchTrigger{
				URL:    appContext.Config.Trigger.IPFSSearch.QueryURL,
				Period: 30 * time.Second,
			}
			go func() {
				if err := t.Start(ctx, cidChan); err != nil {
					log.Ctx(ctx).Fatal().Err(err).Msg("Failed to start IPFS-Search trigger")
				}
				log.Ctx(ctx).Info().Msg("IPFS-Search trigger stopped")
			}()
		}
		go func() {
			for {
				select {
				case <-ctx.Done():
					log.Ctx(ctx).Info().Msg("Stopping cid channel")
					return
				case c := <-cidChan:
					log.Ctx(ctx).Info().Str("cid", c.String()).Msg("Received CID from trigger")
					err = store.CreateExecution(ctx, uuid.NewString(), c.String())
					if err != nil {
						log.Ctx(ctx).Error().Err(err).Msg("Failed to create execution from trigger")
					}
				}
			}
		}()

		// This is how you set up a basic Gorilla router
		r := mux.NewRouter()

		// Use our validation middleware to check all requests against the
		// OpenAPI schema.
		r.Use(middleware.OapiRequestValidator(swagger))

		api.HandlerFromMuxWithBaseURL(store, r, baseURL)

		s := &http.Server{
			Handler: r,
			Addr:    fmt.Sprintf("0.0.0.0:%d", appContext.Config.Port),
		}

		log.Ctx(ctx).Info().Int("port", appContext.Config.Port).Msg("Starting HTTP server")
		go func() {
			if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
		return err
	}
}
