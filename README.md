# go-kv

A TCP key-value store with a custom text protocol and AOF persistence.

## Overview

go-kv accepts commands over a raw TCP connection using a custom line-based protocol. Data is persisted to an append-only file at `~/.go-kv/data.aof` and replayed on startup to restore the statee.

Two binaries:

- `cmd/server` — the TCP server
- `cmd/cli` — the command-line client

## Running the Server

```bash
go run ./cmd/server/main.go
```

Set environment variables in `cmd/server/.env`:

| Variable  | Description                  |
| --------- | ---------------------------- |
| `PORT`    | TCP port to listen on        |
| `APP_ENV` | `development` / `production` |

## CLI Commands

```bash
go run ./cmd/cli/main.go <command> [arguments]

Commands:
  ping                       Check if the server is alive
  get <key>                  Get the value of a key
  set -k <key> -v <value>    Set a key-value pair
  delete <key>               Delete a key
  expire -k <key> -t <sec>   Set a TTL on a key in seconds
```

### Examples

```bash
go run ./cmd/cli/main.go ping
# PONG

go run ./cmd/cli/main.go set -k name -v kelvin
# OK

go run ./cmd/cli/main.go get lastName
# Kelvin

go run ./cmd/cli/main.go expire -k name -t 60
# OK

go run ./cmd/cli/main.go delete name
# OK
```

## Protocol

Commands are sent as plain text terminated with `\r\n`. See [protocol.md](protocol.md) for the full spec.

| Response prefix | Meaning                              |
| --------------- | ------------------------------------ |
| `+OK`           | Command succeeded                    |
| `+PONG`         | Response to PING                     |
| `$<len>`        | Value follows on the next line (GET) |
| `-NOTFOUND`     | Key does not exist                   |
| `-INVALID`      | Malformed or unrecognised command    |
