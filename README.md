# why

> Explains command-line errors in plain language and suggests safe diagnostics.

[![CI](https://github.com/m5rcel-vibecodes/why/actions/workflows/ci.yml/badge.svg)](https://github.com/m5rcel-vibecodes/why/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.22%2B-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Zero Network / Zero AI](https://img.shields.io/badge/offline-100%25%20deterministic-brightgreen.svg)](#privacy--safety)

`why` is a high-performance command-line utility written in pure Go. It answers the question every developer and system administrator asks: *"Why did this command or service fail, what does it mean, and what safe check should I run next?"*

`why` functions as an entirely **local, deterministic knowledge base**. It does not phone home, requires zero network connectivity, and contains **no AI or cloud dependencies**—delivering instant, reliable, hallucination-free explanations.

---

## Key Features

- **100% Local & Offline**: All 19 diagnostic domains (79+ core error rules) are embedded directly into a single static binary. Zero network calls, zero telemetry, zero AI.
- **Log File & Stream Inspection (`why log`)**: Scan application logs, system logs, or piped journalctl streams. Automatically aggregates distinct error patterns, reports frequencies, highlights line numbers, and provides consolidated diagnostic steps.
- **Deterministic Multi-Stage Matching**: Evaluates canonical errors, captures dynamic variables via regex (files, ports, hostnames, usernames), resolves error codes (HTTP, Linux errno, exit codes, compiler codes), and falls back to fuzzy matching for typos.
- **Safety-First Diagnostics**: Only non-destructive inspection steps are suggested by default. Potentially dangerous actions are explicitly flagged. `why` **never automatically executes commands**.
- **Confidence Separation**: Strictly delineates output into `CONFIRMED FROM INPUT`, `LIKELY CAUSE`, `POSSIBLE CAUSES`, and `NEXT DIAGNOSTIC STEP`—no guesses masquerading as facts.
- **Pipe & Stdin Friendly**: Pipe standard error or terminal logs directly into `why` (`command 2>&1 | why`).
- **Structured JSON**: Ready for scripting and pipeline automation (`why "connection refused" --json` or `why log app.log --json`).
- **Extensible Configuration**: Add internal team or proprietary error rules without recompilation using custom YAML rule directories.

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

Static binaries with `CGO_ENABLED=0` are available for Linux (amd64, arm64), macOS (amd64, arm64), and Windows (amd64).

---

## Usage

### 1. Direct Error Lookup

Explain an error directly:

```bash
why "permission denied"
why "connection refused"
why "command not found"
why "Host key verification failed"
why "CrashLoopBackOff"
```

Quoting is optional for quick lookups:

```bash
why git push rejected
why 404
why EACCES
why exit code 137
why error CS0246
```

### 2. Piping from Commands (stdin)

Capture and explain stderr in real time:

```bash
# Git push failure
git push origin main 2>&1 | why

# Docker startup issues
docker run -p 80:80 nginx 2>&1 | why

# Package manager lock errors
sudo apt update 2>&1 | why
```

---

## Log Inspection (`why log`)

Inspect server, application, or system logs to identify, count, and explain all errors present in the file or stream:

```bash
# Scan a local log file
why log /var/log/nginx/error.log
why log app.log

# Stream and inspect logs from journalctl or docker
journalctl -u postgresql -n 200 | why log -
docker logs my-app 2>&1 | why log -

# Machine-readable JSON summary of all log errors
why log app.log --json
```

### Example Log Inspection Output

```console
$ why log /var/log/app.log

LOG INSPECTION REPORT
Source: /var/log/app.log

Summary:
  • Scanned: 1,420 lines
  • Total errors detected: 14 occurrences
  • Distinct error types:  3

ERRORS DETECTED (ORDERED BY FREQUENCY):

[1] 10x connection refused (Networking)
    Lines: 12, 18, 25, 40, 52, 60, 77, 89, 94, 105
    Sample: dial tcp 127.0.0.1:5432: connect: connection refused
    Likely Cause: No service is actively listening on the destination port, or a local host firewall actively rejected the packet.
    Suggested Checks:
      • nc -zv 127.0.0.1 5432
      • sudo ss -tulpn | grep :5432
      • ping -c 3 127.0.0.1
    Caution: Binding a service to 0.0.0.0 exposes it to all available network interfaces including the public internet.

[2] 3x no space left on device (Linux)
    Lines: 210, 214, 220
    Sample: WARN disk write failed: no space left on device
    Likely Cause: The target partition is 100% full, or all available inode numbers on the filesystem have been exhausted.
    Suggested Checks:
      • df -h
      • df -i
      • lsof +L1
    Caution: Do not run rm -rf / or blindly delete files in /usr, /lib, or /boot.

[3] 1x 502 Bad Gateway (HTTP)
    Lines: 432
    Sample: [error] 1242#0: *8 connect() failed (111: Connection refused) while connecting to upstream
    Likely Cause: The backend upstream application process crashed or is not running.
    Suggested Checks:
      • sudo tail -n 30 /var/log/nginx/error.log
      • systemctl status <backend-service>
```

---

## Example Error Output

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

`why` embeds rules across **19 core systems domains**:

| Domain | Example Covered Errors |
|:---|:---|
| **Linux** | `permission denied`, `command not found`, `no space left on device`, `read-only file system`, `device or resource busy`, `broken pipe`, `killed / oom`, `bad interpreter: No such file or directory`, `fork: retry: Resource temporarily unavailable` |
| **SSH** | `Host key verification failed`, `Permission denied (publickey)`, `Permissions are too open` (unprotected private key), `Connection timed out` |
| **Git** | `non-fast-forward push rejected`, `merge conflict`, `remote permission denied`, `refusing to merge unrelated histories`, `local changes would be overwritten`, `RPC failed; HTTP 413`, `index.lock exists`, `not a git repository` |
| **Docker** | `cannot connect to the Docker daemon`, `port is already allocated`, `container exited code 137 (OOM)`, `pull access denied` |
| **Kubernetes** | `CrashLoopBackOff`, `ImagePullBackOff` / `ErrImagePull`, `OOMKilled` (pod limit exceeded) |
| **Databases** | **PostgreSQL** (`password authentication failed`, `remaining connection slots reserved`), **MySQL** (`Access denied (1045)`, `Cannot connect through socket (2002)`), **Redis** (`OOM command not allowed maxmemory`) |
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
| **Rust** | `error[E0382] use of moved value`, `error[E0502] cannot borrow as mutable` |
| **Java / JVM** | `ClassNotFoundException` / `NoClassDefFoundError`, `OutOfMemoryError: Java heap space`, `UnsupportedClassVersionError` |
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
```

### Environment Variables

| Variable | Description |
|:---|:---|
| `WHY_RULES_DIR` | Directory containing custom rule `.yaml` files |
| `WHY_COLOR` | Color mode (`auto`, `always`, `never`) |
| `WHY_JSON` | If set to `1` or `true`, outputs JSON format |
| `NO_COLOR` | Disables ANSI color escapes per [no-color.org](https://no-color.org) |

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
