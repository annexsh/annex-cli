package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/annexsh/annex/server"
	"github.com/annexsh/annex/uuid"
	"github.com/annexsh/annex/workflowservice"
	"github.com/lmittmann/tint"
	"github.com/temporalio/cli/temporalcli/devserver"
	"golang.org/x/sync/errgroup"
)

const (
	ip         = "127.0.0.1"
	serverPort = 4400
	uiPort     = 5400
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	ctx := context.Background()

	temporalSrv, temporalAddr, err := setupTemporalDevServer()
	if err != nil {
		return err
	}
	defer temporalSrv.Stop()

	errg := new(errgroup.Group)

	uiAddr := fmt.Sprintf("%s:%d", ip, uiPort)

	cfg := server.AllInOneConfig{
		Port:              serverPort,
		CorsOrigins:       []string{"http://" + uiAddr},
		StructuredLogging: false,
		SQLite:            true,
		Nats: server.NatsConfig{
			HostPort: fmt.Sprintf("%s:%d", ip, devserver.MustGetFreePort()),
			Embedded: true,
		},
		Temporal: server.TemporalConfig{
			HostPort:  temporalAddr,
			Namespace: workflowservice.Namespace,
		},
	}

	errg.Go(func() error {
		return server.ServeAllInOne(ctx, cfg)
	})

	errg.Go(func() error {
		uiSrv := newUIServer()
		if uiErr := uiSrv.Start(uiAddr); uiErr != nil {
			return uiErr
		}
		return uiSrv.Close()
	})

	slog.Info("serving UI on http://" + uiAddr)

	return errg.Wait()
}

func setupTemporalDevServer() (*devserver.Server, string, error) {
	port := devserver.MustGetFreePort()
	address := fmt.Sprintf("%s:%d", ip, port)
	srv, err := devserver.Start(devserver.StartOptions{
		FrontendIP:             ip,
		FrontendPort:           port,
		UIIP:                   ip,
		UIPort:                 devserver.MustGetFreePort(),
		Namespaces:             []string{workflowservice.Namespace},
		ClusterID:              uuid.NewString(),
		MasterClusterName:      "active",
		CurrentClusterName:     "active",
		InitialFailoverVersion: 1,
		Logger:                 slog.New(tint.NewHandler(os.Stdout, nil)),
		LogLevel:               slog.LevelWarn,
	})
	return srv, address, err
}
