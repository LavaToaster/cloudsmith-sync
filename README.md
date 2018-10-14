Cloudsmith Sync
===============

This tool is to sync your composer repositories from git to Cloudsmith.

## Setup

This package assumes that there is a running SSH agent and it already has the key
responsible for accessing the repositories.

You may or may not need to manually fetch the dependencies required for this project.

```bash
go get github.com/spf13/viper
go get github.com/spf13/cobra
go get gopkg.in/libgit2/git2go.v27
go get github.com/briandowns/spinner
go get github.com/cloudsmith-io/cloudsmith-api
```

Copy `config.example.yaml` to `config.yaml` and amend to your needs. It should be fairly straight forward. 😁

## Running

Running the sync utility
```bash
$ go run main.go run
```

## Roadmap

- [ ] Add a http server that accepts github webhooks commit and tag events and publishes their respective
    versions to Cloudsmith.
