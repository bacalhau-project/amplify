package cmd

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bacalhau-project/amplify/pkg/analytics"
	"github.com/bacalhau-project/amplify/pkg/api"
	"github.com/bacalhau-project/amplify/pkg/cli"
	"github.com/bacalhau-project/amplify/pkg/dag"
	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/bacalhau-project/amplify/pkg/item"
	"github.com/bacalhau-project/amplify/pkg/queue"
	"github.com/bacalhau-project/amplify/pkg/task"
	"github.com/bacalhau-project/amplify/pkg/triggers"
	"github.com/bacalhau-project/amplify/ui"
	middleware "github.com/deepmap/oapi-codegen/pkg/chi-middleware"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const baseURL = ""

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
				URL: baseURL + "/api",
			},
		}

		// DAG Queue
		log.Ctx(ctx).Info().Int("concurrency", appContext.Config.WorkflowConcurrency).Int("max-waiting", appContext.Config.MaxWaitingWorkflows).Msg("Starting DAG queue")
		dagQueue, err := queue.NewGenericQueue(ctx, appContext.Config.WorkflowConcurrency, appContext.Config.MaxWaitingWorkflows)
		if err != nil {
			return err
		}
		dagQueue.Start()
		defer dagQueue.Stop()

		// Node Queue
		log.Ctx(ctx).Info().Int("concurrency", appContext.Config.NodeConcurrency).Msg("Starting node queue")
		nodeQueue, err := queue.NewGenericQueue(ctx, appContext.Config.NodeConcurrency, 1024)
		if err != nil {
			return err
		}
		nodeQueue.Start()
		defer nodeQueue.Stop()

		// Persistence is where node/queue data is stored
		var persistenceImpl db.Persistence
		if strings.HasPrefix(appContext.Config.DB.URI, "postgres://") ||
			strings.HasPrefix(appContext.Config.DB.URI, "postgresql://") {
			log.Ctx(ctx).Info().Str("uri", appContext.Config.DB.URI).Msg("Persisting data to Postgres")
			persistenceImpl, err = db.NewPostgresDB(appContext.Config.DB.URI)
			if err != nil {
				return err
			}
		} else {
			log.Ctx(ctx).Info().Msg("Persisting in memory only")
			persistenceImpl = db.NewInMemDB()
		}

		// NodeStore stores nodes
		nodeStore, err := dag.NewNodeStore(
			ctx,
			persistenceImpl,
			dag.NewInMemWorkRepository[dag.IOSpec](),
		)
		if err != nil {
			return err
		}

		// TODO: Rename this to dags, and move nodes to nodes
		// TaskFactory creates full dags
		taskFactory, err := task.NewTaskFactory(appContext, nodeQueue, nodeStore)
		if err != nil {
			return err
		}

		// ItemStore stores queue items
		itemStore, err := item.NewItemStore(ctx, persistenceImpl.(db.Queue), nodeStore)
		if err != nil {
			return err
		}

		// QueueRepository interacts with the queue
		queueRepository, err := item.NewQueueRepository(itemStore, dagQueue, taskFactory)
		if err != nil {
			return err
		}

		// AnalyticsRepository manages amplify analytics
		analyticsRepository := analytics.NewAnalyticsRepository(persistenceImpl.(db.Analytics))

		// AmplifyAPI provides the REST API
		amplifyAPI, err := api.NewAmplifyAPI(queueRepository, taskFactory, analyticsRepository)
		if err != nil {
			return err
		}

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
					err = amplifyAPI.CreateExecution(ctx, uuid.New(), c.String())
					if err != nil {
						if err == queue.ErrQueueFull {
							log.Ctx(ctx).Warn().Err(err).Msg("Rate limiting new trigger executions, queue full")
							continue
						} else {
							log.Ctx(ctx).Error().Err(err).Msg("Failed to create execution from trigger")
						}
					}
				}
			}
		}()

		// Create the router
		r := mux.NewRouter()

		if appContext.Config.DisableCORS {
			log.Ctx(ctx).Info().Msg("Disabling CORS")
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Access-Control-Allow-Origin", "*")
					w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
					w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
					next.ServeHTTP(w, r)
				})
			})
		}

		apiRouter := r.PathPrefix(baseURL + "/").Subrouter()

		// Use our validation middleware to check all requests against the
		// OpenAPI schema.
		apiRouter.Use(middleware.OapiRequestValidator(swagger))

		api.HandlerFromMuxWithBaseURL(amplifyAPI, apiRouter, "/api")

		// Add UI handler
		handler := ui.AssetHandler(ctx, baseURL)
		r.PathPrefix(baseURL + "/").Handler(handler)

		host := fmt.Sprintf("0.0.0.0:%d", appContext.Config.Port)
		s := &http.Server{
			Handler: r,
			Addr:    host,
		}

		log.Ctx(ctx).Info().Str("address", fmt.Sprintf("http://%s", host)).Msg("Starting HTTP server")
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
