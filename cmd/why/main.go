package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/m5rcel-vibecodes/why/internal/ai"
	"github.com/m5rcel-vibecodes/why/internal/config"
	"github.com/m5rcel-vibecodes/why/internal/extractor"
	"github.com/m5rcel-vibecodes/why/internal/formatter"
	"github.com/m5rcel-vibecodes/why/internal/kb"
	"github.com/m5rcel-vibecodes/why/internal/matcher"
	"github.com/m5rcel-vibecodes/why/internal/model"
)

var (
	version = "1.0.0"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		jsonFlag    bool
		aiFlag      bool
		colorFlag   string
		configPath  string
		versionFlag bool
		helpFlag    bool
	)

	flag.BoolVar(&jsonFlag, "json", false, "Output explanation in structured JSON format")
	flag.BoolVar(&aiFlag, "ai", false, "Use optional AI model for explanation (opt-in)")
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

	// Collect input from arguments or stdin
	args := flag.Args()
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

	// Handle AI opt-in if requested
	if aiFlag {
		aiClient := ai.NewClient(&cfg.AI)
		aiResult, aiErr := aiClient.Query(context.Background(), query)
		if aiErr != nil {
			if !cfg.JSON {
				fmt.Fprintf(os.Stderr, "%s\n", styler.Red(fmt.Sprintf("AI Explanation Error: %v", aiErr)))
				if matchResult != nil {
					fmt.Fprintf(os.Stderr, "%s\n\n", styler.Dim("Falling back to local knowledge base explanation:"))
				}
			}
		} else {
			matchResult = aiResult
		}
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
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// Check if flag takes a separate argument
			if (arg == "--color" || arg == "-color" || arg == "--config" || arg == "-config") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
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
  <command> 2>&1 | why

EXAMPLES:
  why "permission denied"
  why "connection refused"
  why "command not found"
  why 404
  why "exit code 137"
  git push 2>&1 | why
  docker run nginx 2>&1 | why

FLAGS:
  --json          Output structured JSON for scripts and integrations
  --ai            Opt-in to AI-assisted analysis (requires GEMINI_API_KEY, OPENAI_API_KEY, or local Ollama)
  --color <mode>  Color mode: auto (default), always, never
  --config <path> Path to custom configuration file (~/.config/why/config.yaml)
  -v, --version   Print version information
  -h, --help      Show this help message

KNOWLEDGE BASE:
  Covers Linux, Git, Docker, systemd, OpenRC, Networking, DNS, HTTP, TLS,
  Package Managers (apt, pacman, npm, pip, brew), Node.js, Python, Go, and .NET.
  Runs 100% locally and deterministically without network access.
`
	io.WriteString(os.Stdout, usageText)
}
