package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *Application) Serve(mux *http.ServeMux) error {
	srv := &http.Server{
		Addr:         app.Config.HTTPPort,
		Handler:      app.BuildRoutes(mux),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	shutdownErr := make(chan error)

	go func() {
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		s := <-shutdown
		fmt.Printf("shutting down server with signal %v\n", s)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownErr <- err
		}

		fmt.Println("completing background tasks before shutting down...")
		shutdownErr <- nil
	}()

	fmt.Printf("starting server on port %v\n", app.Config.HTTPPort)

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownErr
	if err != nil {
		return err
	}

	fmt.Printf("stopped server %v\n", app.Config.HTTPPort)

	return nil
}
