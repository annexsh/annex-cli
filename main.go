package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/annexsh/annex/log"
	"github.com/annexsh/annex/server"
	"github.com/annexsh/annex/uuid"
	"github.com/annexsh/annex/workflowservice"
	"github.com/lmittmann/tint"
	"github.com/temporalio/cli/temporalcli/devserver"
)

const (
	ip         = "127.0.0.1"
	serverPort = 4400
	uiPort     = 5400
)

func main() {
	if err := run(); err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Println("stopped development server")
			return
		}
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	temporalSrv, temporalAddr, err := startTemporalDevServer()
	if err != nil {
		return err
	}
	defer temporalSrv.Stop()

	uiAddr := fmt.Sprintf("%s:%d", ip, uiPort)

	errs := make(chan error, 1)

	go func() {
		errs <- server.ServeAllInOne(ctx, server.AllInOneConfig{
			Port:              serverPort,
			CorsOrigins:       []string{"http://" + uiAddr},
			StructuredLogging: false,
			SQLite:            true,
			Nats: server.NatsConfig{
				HostPort: fmt.Sprintf("%s:%d", ip, freePort()),
				Embedded: true,
			},
			Temporal: server.TemporalConfig{
				HostPort:  temporalAddr,
				Namespace: workflowservice.Namespace,
			},
		})
	}()

	logger := log.NewDevLogger()
	logger.Info("serving ui on http://" + uiAddr)
	uiSrv := newUIServer()
	go func() {
		errs <- uiSrv.Start(uiAddr)
	}()
	defer uiSrv.Close()

	select {
	case err = <-errs:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func startTemporalDevServer() (*devserver.Server, string, error) {
	port := freePort()
	address := fmt.Sprintf("%s:%d", ip, port)
	srv, err := devserver.Start(devserver.StartOptions{
		FrontendIP:             ip,
		FrontendPort:           port,
		UIIP:                   ip,
		UIPort:                 freePort(),
		Namespaces:             []string{workflowservice.Namespace},
		ClusterID:              uuid.NewString(),
		MasterClusterName:      "active",
		CurrentClusterName:     "active",
		InitialFailoverVersion: 1,
		Logger:                 slog.New(tint.NewHandler(os.Stdout, nil)),
		LogLevel:               slog.LevelError,
	})
	return srv, address, err
}

func freePort() int {
	return devserver.MustGetFreePort(ip)
}
