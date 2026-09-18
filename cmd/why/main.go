package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/m5rcel-vibecodes/why/internal/config"
	"github.com/m5rcel-vibecodes/why/internal/extractor"
	"github.com/m5rcel-vibecodes/why/internal/formatter"
	"github.com/m5rcel-vibecodes/why/internal/kb"
	"github.com/m5rcel-vibecodes/why/internal/logscan"
	"github.com/m5rcel-vibecodes/why/internal/matcher"
	"github.com/m5rcel-vibecodes/why/internal/mcp"
	"github.com/m5rcel-vibecodes/why/internal/model"
)

var (
	version = "1.1.0"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		jsonFlag    bool
		logFlag     string
		colorFlag   string
		configPath  string
		versionFlag bool
		helpFlag    bool
	)

	flag.BoolVar(&jsonFlag, "json", false, "Output explanation in structured JSON format")
	flag.StringVar(&logFlag, "log", "", "Inspect a log file or stream and aggregate all errors")
	flag.StringVar(&colorFlag, "color", "", "Colorize output: auto, always, never")
	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.BoolVar(&versionFlag, "version", false, "Print version and exit")
	flag.BoolVar(&versionFlag, "v", false, "Print version and exit (shorthand)")
	flag.BoolVar(&helpFlag, "help", false, "Print help and exit")
	flag.BoolVar(&helpFlag, "h", false, "Print help and exit (shorthand)")

	flag.Usage = printUsage
	os.Args = append([]string{os.Args[0]}, normalizeArgs(os.Args[1:])...)
	flag.Parse()

	if versionFlag {
		fmt.Printf("why version %s (%s) built at %s\n", version, commit, date)
		os.Exit(0)
	}

	if helpFlag {
		printUsage()
		os.Exit(0)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
	}

	// Flag overrides config
	if jsonFlag {
		cfg.JSON = true
	}
	if colorFlag != "" {
		cfg.Color = colorFlag
	}

	styler := formatter.NewStyler(formatter.ColorMode(cfg.Color))

	// Load Knowledge Base
	knowledgeBase, err := kb.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading local knowledge base: %v\n", err)
		os.Exit(1)
	}

	if cfg.RulesDir != "" {
		_ = knowledgeBase.LoadCustomDir(cfg.RulesDir)
	}

	ruleMatcher := matcher.New(knowledgeBase)
	args := flag.Args()

	// Check if user requested MCP server: e.g. `why mcp` or `why serve-mcp`
	if len(args) > 0 && (args[0] == "mcp" || args[0] == "serve-mcp") {
		server := mcp.NewServer(knowledgeBase, ruleMatcher, version)
		if err := server.Serve(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "why MCP server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Check if user requested log inspection via subcommand or flag:
	// e.g. `why log <file>` or `why log -` or `why --log <file>`
	isLogCommand := false
	logTarget := logFlag

	if len(args) > 0 && (args[0] == "log" || args[0] == "inspect") {
		isLogCommand = true
		if len(args) > 1 {
			logTarget = args[1]
		} else {
			logTarget = "-"
		}
	} else if logFlag != "" {
		isLogCommand = true
	}

	if isLogCommand {
		report, err := logscan.ScanFile(logTarget, ruleMatcher)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Log inspection error: %v\n", err)
			os.Exit(1)
		}

		if cfg.JSON {
			output, err := logscan.FormatJSONReport(report)
			if err != nil {
				fmt.Fprintf(os.Stderr, "JSON formatting error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(output))
		} else {
			fmt.Print(logscan.FormatTerminalReport(report, styler))
		}
		return
	}

	// Standard error lookup mode
	var query string
	var matchResult *model.MatchResult

	if len(args) > 0 {
		query = strings.Join(args, " ")
		matchResult = ruleMatcher.Match(query)
	} else {
		// Check stdin
		stat, _ := os.Stdin.Stat()
		isPiped := (stat.Mode() & os.ModeCharDevice) == 0
		if isPiped {
			matchResult, query = extractor.ExtractAndMatch(os.Stdin, ruleMatcher)
		}
	}

	query = strings.TrimSpace(query)
	if query == "" {
		printUsage()
		os.Exit(0)
	}

	// Output formatting
	if cfg.JSON {
		output, err := formatter.FormatJSON(matchResult)
		if err != nil {
			fmt.Fprintf(os.Stderr, "JSON formatting error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(output))
		if matchResult == nil {
			os.Exit(1)
		}
		return
	}

	if matchResult == nil {
		fmt.Print(formatter.FormatTerminal(nil, styler))
		os.Exit(1)
	}

	fmt.Print(formatter.FormatTerminal(matchResult, styler))
}

func normalizeArgs(args []string) []string {
	var flags []string
	var pos []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			flags = append(flags, arg)
			// Check if flag takes a separate argument
			if (arg == "--color" || arg == "-color" || arg == "--config" || arg == "-config" || arg == "--log" || arg == "-log") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
		} else {
			pos = append(pos, arg)
		}
	}
	return append(flags, pos...)
}

func printUsage() {
	usageText := `why - Explains command-line errors in plain language and suggests safe diagnostics

USAGE:
  why "<error message>"
  why <error message>
  why log <log-file-path>
  why mcp
  <command> 2>&1 | why
  <command> 2>&1 | why log -

ERROR LOOKUP EXAMPLES:
  why "permission denied"
  why "connection refused"
  why "command not found"
  why 404
  why "exit code 137"
  git push 2>&1 | why
  docker run nginx 2>&1 | why

LOG INSPECTION EXAMPLES:
  why log /var/log/nginx/error.log
  why log app.log
  why log app.log --json
  journalctl -u myapp -n 100 | why log -

FLAGS:
  --json          Output explanation in structured JSON format
  --log <path>    Inspect a log file and aggregate all detected errors
  --color <mode>  Color mode: auto (default), always, never
  --config <path> Path to custom configuration file (~/.config/why/config.yaml)
  -v, --version   Print version information
  -h, --help      Show this help message

KNOWLEDGE BASE:
  Covers Linux, Git, Docker, Kubernetes, Databases (Postgres, MySQL, Redis), SSH,
  systemd, OpenRC, Networking, DNS, HTTP, TLS, Package Managers (apt, pacman, npm, pip, brew),
  Node.js, Python, Go, Rust, Java, and .NET.
  Runs 100% locally and deterministically without network or AI dependencies.
`
	io.WriteString(os.Stdout, usageText)
}
