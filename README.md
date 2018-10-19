Cloudsmith Sync
===============

This tool is to sync your composer repositories from git to Cloudsmith.

## Setup

Get the project

```bash
go get -u github.com/Lavoaster/cloudsmith-sync
```

Change to the directory. (modify if your $GOPATH is different from $HOME/go)
```bash
cd $HOME/go/src/github.com/Lavoaster/cloudsmith-sync/
```

Copy `config.example.yaml` to `config.yaml` and amend to your needs. It should be fairly straight forward. 😁

## Running

Running the sync utility
```bash
$ go run main.go run
```

