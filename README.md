# why

> Explains command-line errors in plain language and suggests safe diagnostics.

[![CI](https://github.com/m5rcel-vibecodes/why/actions/workflows/ci.yml/badge.svg)](https://github.com/m5rcel-vibecodes/why/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.22%2B-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Zero Network](https://img.shields.io/badge/network-zero%20dependency-brightgreen.svg)](#privacy--safety)

`why` is a high-performance command-line utility written in pure Go. It answers the question every developer and system administrator asks: *"Why did this command fail, what does it mean, and what safe check should I run next?"*

`why` functions primarily as a **deterministic local knowledge base** that runs entirely offline. It does not phone home, does not require internet access, and does not require AI.

---

## Key Features

- **100% Local & Private**: All 14 diagnostic domains are embedded directly into a single static binary. No telemetry, no external calls.
- **Deterministic Multi-Stage Matching**: Matches canonical errors, captures dynamic variables via regex (files, ports, hostnames), resolves error codes (HTTP, errno, exit codes), and falls back to fuzzy matching for typos.
- **Safety-First Diagnostics**: Only non-destructive inspection steps are suggested by default. Dangerous actions are explicitly flagged with warnings. `why` **never automatically executes commands**.
- **Confidence Separation**: Separates findings into `CONFIRMED FROM INPUT`, `LIKELY CAUSE`, `POSSIBLE CAUSES`, and `NEXT DIAGNOSTIC STEP`—no guesses masquerading as facts.
- **Pipe & Log Friendly**: Pipe standard error or terminal logs directly into `why` (`command 2>&1 | why`).
- **Structured JSON**: Script and automation ready (`why "connection refused" --json`).
- **Extensible Configuration**: Add internal company error rules without recompilation using custom YAML rule directories.
- **Optional AI Integration**: Opt-in only via `--ai` flag, clearly marked when invoked, supporting Gemini, OpenAI, or local Ollama.

---

## Installation

### From Source (Go 1.22+)

```bash
git clone https://github.com/m5rcel-vibecodes/why.git
cd why
go install ./cmd/why
```

Or build locally:

```bash
make build
# Binary created at ./bin/why
```

### Pre-built Cross-Platform Binaries

Download static binaries for Linux, macOS, and Windows from the [GitHub Releases](https://github.com/m5rcel-vibecodes/why/releases) page.

---

## Usage

### Direct Error Lookup

```bash
why "permission denied"
why "connection refused"
why "command not found"
```

Quoting is optional for quick lookups:

```bash
why git push rejected
why 404
why EACCES
why exit code 137
```

### Piping from Commands (stdin)

Capture and explain stderr in real time:

```bash
# Git push failure
git push origin main 2>&1 | why

# Docker startup issues
docker run -p 80:80 nginx 2>&1 | why

# Package manager lock errors
sudo apt update 2>&1 | why
```

### JSON Output

For CI/CD pipelines, IDE plugins, and automation:

```bash
why "connection refused" --json
```

Output:
```json
{
  "match_type": "exact",
  "confidence_score": 1,
  "confidence_level": "HIGH",
  "matched_input": "connection refused",
  "confirmed_from_input": [
    "Direct error match: connection refused"
  ],
  "likely_cause": "No service is actively listening on the destination port...",
  "diagnostic_steps": [
    {
      "command": "nc -zv <host> <port>",
      "description": "Test basic TCP port connectivity"
    }
  ]
}
```

---

## Example Output

### Linux Permission Error with Parameter Extraction

```console
$ why "bash: /var/log/nginx/access.log: Permission denied"

WHY DID THIS HAPPEN?

Error:
  permission denied (Linux)

Usually means:
  The process does not have sufficient permission to access the requested resource.

CONFIRMED FROM INPUT:
  • Target resource: /var/log/nginx/access.log
  • Operating system access check failed (EACCES / code 13)

LIKELY CAUSE:
  The current user or process lacks read, write, or execute mode bits or ownership for the target path.

POSSIBLE CAUSES:
  • Incorrect ownership (file or directory owned by root or another user)
  • Incorrect file permissions (mode bits lack r/w/x for user or group)
  • Filesystem restrictions (mounted read-only, noexec, or nosuid)
  • Parent directory lacks execute (+x) permission for directory traversal
  • Security policies (SELinux denying context, AppArmor profile restrictions)

CHECK / NEXT DIAGNOSTIC STEP:

  ls -ld /var/log/nginx/access.log
    Inspect file ownership and permission bits

  id
    Check your current user UID and assigned groups

  namei -l /var/log/nginx/access.log
    Trace permissions on every parent directory leading to the target

POTENTIAL SOLUTIONS:

  • Grant execution or read/write permissions to the current user or group
    chmod u+rx /var/log/nginx/access.log

  • Change ownership if you own or manage the file
    chown $USER:$USER /var/log/nginx/access.log

WARNING:
  Do not blindly use sudo before determining the actual cause.
  Never run chmod 777 or recursive chown -R on system directories.
```

---

## Knowledge Database Coverage

`why` embeds rules across **14 core systems domains**:

| Domain | Example Covered Errors |
|:---|:---|
| **Linux** | `permission denied`, `command not found`, `no space left on device`, `read-only file system`, `device or resource busy`, `broken pipe`, `killed / oom`, `too many open files` |
| **Git** | `non-fast-forward push rejected`, `merge conflict`, `remote permission denied`, `refusing to merge unrelated histories`, `index.lock exists`, `not a git repository` |
| **Docker** | `cannot connect to the Docker daemon`, `port is already allocated`, `container exited code 137 (OOM)`, `pull access denied` |
| **systemd** | `Unit not found`, `status=203/EXEC`, `status=1/FAILURE`, `unit is masked`, `Start request repeated too quickly` |
| **OpenRC** | `service does not exist`, `service has crashed`, `status: crashed` |
| **Networking** | `connection refused` (ECONNREFUSED), `connection timed out` (ETIMEDOUT), `address already in use` (EADDRINUSE), `connection reset by peer` |
| **DNS** | `NXDOMAIN`, `SERVFAIL`, `temporary failure in name resolution` (EAI_AGAIN), `could not resolve host` |
| **HTTP** | `400 Bad Request`, `401 Unauthorized`, `403 Forbidden`, `404 Not Found`, `429 Too Many Requests`, `500 Internal Server Error`, `502 Bad Gateway`, `504 Gateway Timeout` |
| **TLS / SSL** | `certificate has expired`, `certificate signed by unknown authority`, `hostname mismatch` |
| **Package Managers** | `apt` (dpkg lock), `pacman` (database lock), `pip` (externally-managed-environment), `npm` (ERESOLVE peer conflict) |
| **Node.js** | `ERR_MODULE_NOT_FOUND`, `JavaScript heap out of memory`, `ERR_REQUIRE_ESM` |
| **Python** | `ModuleNotFoundError`, `IndentationError`, `RecursionError` |
| **Go** | `nil pointer dereference`, `all goroutines are asleep (deadlock)`, `updates to go.mod needed`, `imported and not used` |
| **.NET** | `CS0246 type or namespace not found`, `System.NullReferenceException`, `MSB4019 imported project not found` |

---

## Custom Rules (Extensibility)

You can extend `why` with internal team or proprietary error rules without re-compiling.

Place YAML files into `~/.config/why/rules.d/*.yaml`:

```yaml
rules:
  - id: internal-db-auth-failed
    error: internal db authentication failure
    category: Internal Database
    meaning: The application failed to authenticate against the internal cluster.
    exact_matches:
      - "internal db auth failed"
    patterns:
      - '(?i)failed to authenticate user "(?P<user>[^"]+)" against cluster (?P<cluster>\S+)'
    likely_cause: The database token expired or the role has not been provisioned in Vault.
    possible_causes:
      - Expired Vault lease token
      - Database user credential rotated
    diagnostic_steps:
      - command: "vault token lookup"
        description: "Check current Vault authentication status and remaining lease duration"
    potential_solutions:
      - description: "Renew database credentials using Vault"
        command: "vault login"
    warnings:
      - "Never log plaintext database connection strings containing passwords."
```

---

## Configuration

Configuration file location: `~/.config/why/config.yaml`

```yaml
# Directory containing custom rules
rules_dir: "~/.config/why/rules.d"

# Color mode: auto, always, never
color: "auto"

# Output JSON by default (default: false)
json: false

# Optional AI settings (used only with --ai)
ai:
  provider: "ollama"       # ollama, gemini, or openai
  model: "llama3"
  base_url: "http://localhost:11434"
```

### Environment Variables

| Variable | Description |
|:---|:---|
| `WHY_RULES_DIR` | Directory containing custom rule `.yaml` files |
| `WHY_COLOR` | Color mode (`auto`, `always`, `never`) |
| `WHY_JSON` | If set to `1` or `true`, outputs JSON format |
| `NO_COLOR` | Disables ANSI color escapes per [no-color.org](https://no-color.org) |
| `GEMINI_API_KEY` | API key for optional `--ai` Gemini analysis |
| `OPENAI_API_KEY` | API key for optional `--ai` OpenAI analysis |

---

## Optional AI Integration (`--ai`)

`why` is designed from the ground up to be independent of cloud AI. However, for unknown or bespoke errors, you can explicitly opt in to an AI analysis:

```bash
why "strange custom kernel failure" --ai
```

- **Strictly Opt-In**: AI is never called unless `--ai` is passed.
- **No Silent Uploads**: Your terminal commands and errors are never transmitted automatically.
- **Clear Attribution**: Output is explicitly marked:
  ```
  [AI GENERATED EXPLANATION] (Generated via AI model; verify all suggestions before executing)
  ```
- **Local AI Supported**: Default AI provider is local [Ollama](https://ollama.ai) (`llama3`), allowing 100% offline AI queries without cloud APIs.

---

## Contributing

1. Fork and clone the repository.
2. Ensure you have Go 1.22+ installed.
3. Run test suite:
   ```bash
   make test
   make lint
   ```
4. Submit a pull request.

---

## License

MIT License. See [LICENSE](LICENSE) for details.
