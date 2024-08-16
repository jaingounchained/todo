package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"

	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/jaingounchained/todo/api"
	db "github.com/jaingounchained/todo/db/sqlc"
	_ "github.com/jaingounchained/todo/docs"
	"github.com/jaingounchained/todo/gapi"
	"github.com/jaingounchained/todo/pb"
	storage "github.com/jaingounchained/todo/storage"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	localStorage "github.com/jaingounchained/todo/storage/local_directory"
	"github.com/jaingounchained/todo/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

//	@title			Todo API
//	@version		1.0
//	@description	A todo management service API in go which supports attachments

//	@contact.name	Bhavya Jain

//	@host		localhost:8080
//	@BasePath	/

// @securityDefinitions.apikey	AccessTokenAuth
// @in							header
// @name						Authorization
// @description				To access todos and attachments
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	defer stop()

	// Load config
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// Logging setup
	if config.Environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	log.Info().Msg("logger setup")

	// Setup Postgres DB
	dbStore := setupDBConn(ctx, config)

	// Setup Storage
	storage := setupStorage(ctx, config)

	waitGroup, ctx := errgroup.WithContext(ctx)

	// runGatewayServer(config, store)
	runGRPCServer(ctx, waitGroup, config, dbStore)
	startHTTPGinServer(ctx, waitGroup, config, dbStore, storage)

	err = waitGroup.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("error from wait group")
	}
}

func setupDBConn(
	ctx context.Context,
	config util.Config,
) db.Store {
	// Setup DB connection
	connPool, err := pgxpool.New(ctx, config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to the postgreSQL db")
	}

	if err := connPool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to ping the db")
	}

	// run db migration
	runDBMigration(config.MigrationURL, config.DBSource)

	return db.NewStore(connPool)
}

func runDBMigration(migrationURL, dbSource string) {
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create new migrate instance")
	}

	if err := migration.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal().Err(err).Msg("failed to run migrate up")
	}

	log.Info().Msg("db migrated successfully")
}

func setupStorage(
	ctx context.Context,
	config util.Config,
) storage.Storage {
	// Setup file storage
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to calculate current working directory")
	}

	var storage storage.Storage
	switch config.StorageType {
	case "LOCAL":
		localStoragePath := filepath.Join(cwd, config.LocalStorageDirectory)

		storage, err = localStorage.New(localStoragePath)
		if err != nil {
			log.Fatal().Err(err).Msg("cannot setup file storage for the local storageType")
		}
	case "S3":
		log.Fatal().Msg("s3 file storage unimplemented")
	default:
		log.Fatal().Msg("invalid file storage type chosen")
	}

	return storage
}

func runGRPCServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	store db.Store,
) {
	server, err := gapi.NewGRPCServer(config, store)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create grpc server")
	}

	grpcLogger := grpc.UnaryInterceptor(gapi.GRPCLogger)
	grpcServer := grpc.NewServer(grpcLogger)
	pb.RegisterTodoServer(grpcServer, server)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create listener")
	}

	waitGroup.Go(func() error {
		log.Info().Msgf("start grpc server at %s", listener.Addr().String())

		err = grpcServer.Serve(listener)
		if err != nil {
			if errors.Is(err, grpc.ErrServerStopped) {
				return nil
			}
			log.Error().Err(err).Msg("grpc server failed to serve")
			return err
		}

		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Info().Msg("graceful shutdown grpc server")

		grpcServer.GracefulStop()
		log.Info().Msg("grpc server is stopped")

		return nil
	})
}

func runGatewayServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	store db.Store,
) {
	server, err := gapi.NewGRPCServer(config, store)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create grpc server")
	}

	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	})

	grpcMux := runtime.NewServeMux(jsonOption)

	err = pb.RegisterTodoHandlerServer(ctx, grpcMux, server)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot register handler server")
	}

	mux := http.NewServeMux()
	mux.Handle("/", grpcMux)

	handler := gapi.HTTPLogger(mux)

	httpServer := &http.Server{
		Handler: handler,
		Addr:    config.HTTPServerAddress,
	}

	waitGroup.Go(func() error {
		log.Info().Msgf("start HTTP gateway server at %s", httpServer.Addr)
		err = httpServer.ListenAndServe()
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			log.Error().Err(err).Msg("HTTP gateway server failed to serve")
			return err
		}

		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Info().Msg("graceful shutdown HTTP gateway server")

		err := httpServer.Shutdown(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("failed to shutdown HTTP gateway server")
			return err
		}

		log.Info().Msg("HTTP gateway server is stopped")
		return nil
	})
}

func startHTTPGinServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	store db.Store,
	storage storage.Storage,
) {
	// Initializing the http server
	server, err := api.NewGinHandler(config, store, storage)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create http server")
	}

	httpServer := &http.Server{
		Handler: server.Router,
		Addr:    config.HTTPServerAddress,
	}

	waitGroup.Go(func() error {
		log.Info().Msgf("start HTTP server at %s", httpServer.Addr)
		err = httpServer.ListenAndServe()
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			log.Error().Err(err).Msg("HTTP server failed to serve")
			return err
		}

		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Info().Msg("graceful shutdown HTTP server")

		err := httpServer.Shutdown(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("failed to shutdown HTTP server")
			return err
		}

		log.Info().Msg("HTTP server is stopped")
		return nil
	})
}

func applicationShutdown(done <-chan os.Signal, httpServer *http.Server, connPool *pgxpool.Pool, storage storage.Storage) {
	<-done
	log.Info().Msg("shutting down server...")

	// Server has 5 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close all database connections
	go connPool.Close()

	// Close storage connections
	go storage.CloseConnection(context.Background())

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server exiting...")
}
