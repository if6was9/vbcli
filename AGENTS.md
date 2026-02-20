# AGENTS.md

This file defines project-specific guidance for coding agents working in `vbcli`.

## Project Purpose

`vbcli` is a Go CLI for sending content to Vestaboard via:

- Vestaboard Cloud API (`https://cloud.vestaboard.com/`)
- Vestaboard VBML API (`https://vbml.vestaboard.com/compose`)

## Tech Stack

- Language: Go
- CLI framework: Cobra

## Current CLI Contract

Top-level commands:

- `vbcli send-raw <characters-json|->`
- `vbcli send <message|->`
- `vbcli clear`
- `vbcli get`

Global flags:

- `-v, --verbose`

`send` flags:

- `-m, --model` (`flagship` or `note`)
- `-a, --align` (`top`, `center`, `bottom`)
- `-j, --justify` (`left`, `center`, `right`, `justified`)

`clear` supports the same layout/model flags as `send`.

Notes:

- `-` means read full input from stdin.
- `text` command does not exist (it was removed intentionally).

## Environment Variables

Required:

- `VESTABOARD_TOKEN` for Cloud API auth, sent as `X-vestaboard-token`.

Optional:

- `VESTABOARD_MODEL` fallback for `send --model` when `--model` is not provided.

## API Behavior Requirements

Cloud API write call:

- Endpoint: `POST https://cloud.vestaboard.com/`
- Content type: `application/json`
- Payload shape for `send-raw` and final `send` step:
  - `{"characters": [[...], ...]}`

Cloud API read call:

- Endpoint: `GET https://cloud.vestaboard.com/`
- Prints response JSON directly to stdout for `get`.
- `get --layout` prints only the `currentMessage.layout` string.

VBML compose call (`send` command):

- Endpoint: `POST https://vbml.vestaboard.com/compose`
- Content type: `application/json`
- Payload shape:
  - `components[0].template` from CLI input
  - `components[0].style.align` and `components[0].style.justify`
  - top-level `style.height=3` and `style.width=15` when model is `note`

Response handling:

- Cloud API `409 Conflict` is treated as success (no non-zero exit).
- VBML responses may be either:
  - `{"characters":[[...], ...]}`
  - `[[...], ...]`

## Input Processing Rules

For `send` input:

1. Decode escaped sequences (for example `\n` -> newline).
2. Substitute named `{alias}` tokens to numeric codes (for example `{green}` -> `{66}`).
3. Preserve VBML `{{...}}` expressions as-is.

For `clear`:

- Behavior must remain equivalent to `vbcli send ''`.

For `send-raw` input:

- Must parse as JSON array of arrays of integers.

## Verbose Logging Rules

When `--verbose` is enabled, log for each HTTP request:

- Request URL
- Pretty-printed request JSON
- Response URL
- HTTP status code
- Pretty-printed response JSON (or `<empty>`)

## Development Expectations

- Keep changes idiomatic Go and small in scope.
- Preserve existing command behavior unless explicitly requested.
- Update tests with behavior changes.
- Run `go test ./...` after edits.
- Keep `README.md` user-facing; keep this file agent-facing.
