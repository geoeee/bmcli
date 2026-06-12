# bmcli

`bmcli` is the Bastion Management CLI. This repository currently contains the
initial Go command skeleton with help and version commands.

## Prerequisites

- Go 1.22 or newer

## Usage

```sh
go run ./cmd/bmcli --help
go run ./cmd/bmcli help
go run ./cmd/bmcli version
```

Build a local binary:

```sh
go build -o bin/bmcli ./cmd/bmcli
./bin/bmcli version
```

Release builds can inject version metadata with linker flags:

```sh
go build -ldflags "-X github.com/geoeee/bmcli/internal/cli.Version=1.0.0 -X github.com/geoeee/bmcli/internal/cli.Commit=$(git rev-parse --short HEAD) -X github.com/geoeee/bmcli/internal/cli.Date=$(date -u +%Y-%m-%d)" -o bin/bmcli ./cmd/bmcli
```

## Development

```sh
gofmt -w ./cmd ./internal
go vet ./...
go test ./...
go build ./...
```
