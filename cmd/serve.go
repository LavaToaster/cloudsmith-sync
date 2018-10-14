package cmd

import (
	"context"
	"fmt"
	"github.com/Lavoaster/cloudsmith-sync/cloudsmith"
	"github.com/Lavoaster/cloudsmith-sync/webhooks"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"gopkg.in/go-playground/webhooks.v5/github"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Runs a server that listens for GitHub webhooks",
	Run: func(cmd *cobra.Command, args []string) {
		router := mux.NewRouter()

		hook, err := github.New(github.Options.Secret(config.WebhookSecret))
		exitOnError(err)

		webhooks.Hook = hook

		router.HandleFunc("/webhooks/github", webhooks.HandleGithubWebhook).Methods("POST")

		webhooks.Client = cloudsmith.NewClient(config.ApiKey)
		webhooks.Config = config

		srv := &http.Server{
			Addr: config.Server,

			// Good practice to set timeouts to avoid Slowloris attacks.
			WriteTimeout: time.Second * 15,
			ReadTimeout:  time.Second * 15,
			IdleTimeout:  time.Second * 60,
			Handler:      router, // Pass our instance of gorilla/mux in.
		}

		go func() {
			fmt.Println("Server listening on " + srv.Addr)

			if err := srv.ListenAndServe(); err != nil {
				exitOnError(err)
			}
		}()

		c := make(chan os.Signal, 1)

		// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
		// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
		signal.Notify(c, os.Interrupt)

		// Block until we receive our signal.
		<-c

		// Create a deadline to wait for.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		// Doesn't block if no connections, but will otherwise wait
		// until the timeout deadline.

		srv.Shutdown(ctx)

		// Optionally, you could run srv.Shutdown in a goroutine and block on
		// <-ctx.Done() if your application should wait for other services
		// to finalize based on context cancellation.
		fmt.Println("shutting down")
		os.Exit(0)
	},
}
