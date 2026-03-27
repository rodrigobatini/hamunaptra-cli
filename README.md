<div align="center">

# Hamunaptra CLI

### Your stack already has great CLIs. You deserve **one more** that ties cost and usage together.

A **Go** binary for people who live in **Vercel**, **Supabase**, **Neon**, **Render**, and the terminal — with **`doctor`** to validate tooling, **browser-based device login**, and commands that talk to the **Hamunaptra API** when you have it running.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev/)

[**Install**](#install) · [**Commands**](#commands) · [**Why**](#why-this-cli-exists) · [**Ecosystem**](#where-it-fits)

</div>

---

## Why this CLI exists

Every vendor ships a solid CLI. The pain is elsewhere: the **full picture** — how much you spend, where, and whether the tools you think are installed actually work — stays **fragmented** across tabs and dashboards.

Hamunaptra does **not** ask you to abandon `vercel` or the flows you trust. It asks for this:

> **Let me be the calm entrypoint:** one terminal command, fast checks, and an API-backed path when you want **sync**, **reports**, **anomalies**, and **ask** without losing your mind.

If that sounds like the tool **you** wished you had on your last project — this repo is for you. **Star it**, open issues, ship PRs.

---

## Install

### Option A: npx (no Go required)

```bash
npx -y hamunaptra-cli --help
```

Install globally with npm if you prefer:

```bash
npm i -g hamunaptra-cli
hamunaptra --help
```

### Option B: Go install

If you prefer native Go tooling:

```bash
go install github.com/rodrigobatini/hamunaptra-cli/cmd/hamunaptra@latest
```

Sanity check:

```bash
hamunaptra version
hamunaptra doctor
```

`doctor` probes vendor CLIs on your machine (with timeouts so nothing hangs forever).

### Build from source

```bash
git clone https://github.com/rodrigobatini/hamunaptra-cli.git
cd hamunaptra-cli
go build -o hamunaptra ./cmd/hamunaptra
./hamunaptra doctor
```

> **How npm works here:** the npm package is a launcher that downloads the correct prebuilt Go binary for your OS/arch from GitHub Releases.

### Maintainers: release asset names

For npm/npx installs to work, each GitHub release should publish binaries with these names:

- `hamunaptra-linux-amd64`
- `hamunaptra-linux-arm64`
- `hamunaptra-darwin-amd64`
- `hamunaptra-darwin-arm64`
- `hamunaptra-windows-amd64.exe`
- `hamunaptra-windows-arm64.exe`

---

## Configuration

| What | Where |
|------|--------|
| Token after `login` | `~/.config/hamunaptra/config.json` |
| API base URL | Env **`HAMUNAPTRA_API`** (default `http://127.0.0.1:8081`) |
| Local project | **`hamunaptra.yaml`** (created by `init`) |

```bash
export HAMUNAPTRA_API=https://api.example.com
hamunaptra login
```

---

## Commands

| Command | Purpose |
|---------|---------|
| `login` | Browser device flow; after you approve on `/login/cli`, stores `ham_…` token |
| `logout` | Remove local token |
| `whoami` | `GET /v1/me` |
| `init` | Create project on the API + write `hamunaptra.yaml` |
| `connect` | e.g. `connect vercel` — register a provider on the project |
| `sync` | Trigger server-side sync |
| `report` | Cost series (`--json`, `--from`, `--to`) |
| `anomalies` | List anomalies |
| `ask` | Natural-language question (aggregates stay on the server) |
| `doctor` | Check vendor CLIs on `PATH` |

---

## Where it fits

```
   hamunaptra CLI  ──────▶  Hamunaptra API (Go)  ──────▶  Postgres
        │                           ▲
        │                           │
        └──── device login ──────────┘
                    via browser (web app + Clerk)
```

- **Web app & docs** (marketing, `/app`, `/docs/`): [**hamunaptra**](https://github.com/rodrigobatini/hamunaptra) monorepo.
- **Go API**: typically a separate private or self-hosted service (Postgres, `/v1`, optional Stripe, MCP).

This repository is the **public, installable** face of the product.

---

## Contributing

Issues and PRs are welcome. For larger changes, open an issue first so we can align on design.

---

<div align="center">

**If this saves you one night of tab-hopping, it paid off.**  
Give the repo a star so more people can find it.

</div>
