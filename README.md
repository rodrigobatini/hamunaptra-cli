# hamunaptra-cli

Hamunaptra CLI (Go): autenticação no **browser** (device flow), projeto local `hamunaptra.yaml`, conexões de providers, sync, relatórios, anomalias e `ask`.

## Config

- Config file: `~/.config/hamunaptra/config.json` (gravado por `login`).
- Env: `HAMUNAPTRA_API` (default `http://127.0.0.1:8081`).

## Comandos

| Comando | Descrição |
|---------|-----------|
| `login` | Inicia fluxo browser; guarda token `ham_…` |
| `logout` | Remove token local |
| `whoami` | Chama `GET /v1/me` |
| `init` | Cria projeto na API + `hamunaptra.yaml` |
| `connect` | `connect vercel` (regista provider no projeto) |
| `sync` | Dispara sync demo no servidor |
| `report` | Série de custos (`--json`, `--from`, `--to`) |
| `anomalies` | Lista anomalias simples |
| `ask` | Pergunta (agregados só no backend) |
| `doctor` | Verifica CLIs locais dos providers |

## Build

```bash
go build -o hamunaptra ./cmd/hamunaptra
```

## Install (repo público)

```bash
go install github.com/rodrigobatini/hamunaptra-cli/cmd/hamunaptra@latest
```

Marketing: [hamunaptra](https://github.com/rodrigobatini/hamunaptra) · API Go: ver repositório/pasta `hamunaptra-server` no workspace.
