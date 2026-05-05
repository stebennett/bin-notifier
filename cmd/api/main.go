package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stebennett/bin-notifier/pkg/api"
	"github.com/stebennett/bin-notifier/pkg/config"
	"github.com/stebennett/bin-notifier/pkg/store"
)

type app struct {
	server   *http.Server
	store    *store.Store
	Listener net.Listener
}

func (a *app) Close() error {
	if a.server != nil {
		_ = a.server.Shutdown(context.Background())
	}
	if a.store != nil {
		return a.store.Close()
	}
	return nil
}

func (a *app) ServeOn(l net.Listener) error {
	if err := a.server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func newApp() (*app, error) {
	configPath := envOr("BN_API_CONFIG_FILE", "")
	dbPath := envOr("BN_API_DB_PATH", "/var/lib/bin-notifier/cache.db")
	listenAddr := envOr("BN_API_LISTEN_ADDR", ":8080")
	readToken := os.Getenv("BN_API_READ_TOKEN")
	writeToken := os.Getenv("BN_API_WRITE_TOKEN")

	if configPath == "" {
		return nil, errors.New("BN_API_CONFIG_FILE is required")
	}
	if readToken == "" || writeToken == "" {
		return nil, errors.New("BN_API_READ_TOKEN and BN_API_WRITE_TOKEN are required")
	}

	cfg, err := config.LoadConfigForMCP(configPath)
	if err != nil {
		return nil, err
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, err
	}
	srv, err := api.NewServer(api.Options{
		Config: cfg, Store: st, ReadToken: readToken, WriteToken: writeToken,
	})
	if err != nil {
		st.Close()
		return nil, err
	}

	httpSrv := &http.Server{
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		st.Close()
		return nil, err
	}
	return &app{server: httpSrv, store: st, Listener: listener}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// Allow `-c` as a convenience flag mirroring other binaries.
	fs := flag.NewFlagSet("bin-notifier-api", flag.ContinueOnError)
	cfgFlag := fs.String("c", "", "path to YAML config file")
	_ = fs.Parse(os.Args[1:])
	if *cfgFlag != "" {
		_ = os.Setenv("BN_API_CONFIG_FILE", *cfgFlag)
	}

	a, err := newApp()
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		_ = a.server.Shutdown(context.Background())
	}()

	log.Printf("bin-notifier-api listening on %s", a.Listener.Addr())
	if err := a.ServeOn(a.Listener); err != nil {
		log.Fatal(err)
	}
}
