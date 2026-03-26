# hamunaptra-cli

Hamunaptra command-line tool: orchestrates official provider CLIs (Vercel, Supabase, Neon, Render, …) for local cost and usage visibility.

The marketing site and pricing live in [`hamunaptra`](https://github.com/rodrigobatini/hamunaptra).

## Requirements

- Go 1.22+ (to build from source)

## Build

```bash
go build -o hamunaptra ./cmd/hamunaptra
./hamunaptra version
./hamunaptra doctor
```

## Install with Go (when this repo is public)

```bash
go install github.com/rodrigobatini/hamunaptra-cli/cmd/hamunaptra@latest
```

## Commands

| Command   | Description                                      |
|-----------|--------------------------------------------------|
| `version` | Print version                                   |
| `doctor`  | Check provider CLIs on PATH                     |
| `sync`    | Stub — future aggregate sync                    |
| `report`  | Stub — future report after sync                 |

## Develop

```bash
go test ./...
```
