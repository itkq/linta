package linta

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

func CreateApp() *cli.App {
	app := &cli.App{
		Name:     "linta",
		Usage:    "A linter for GitHub Actions' permissions",
		Commands: commands(),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Enable debug mode",
				Value: false,
			},
		},
	}
	return app
}

func commands() []*cli.Command {
	return []*cli.Command{
		{
			Name:   "init",
			Usage:  "Initialize config file",
			Action: initAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "output-path",
					Usage:   "Write configuration to `PATH`",
					Aliases: []string{"o"},
				},
				&cli.BoolFlag{
					Name:  "overwrite",
					Usage: "Overwrite existing configuration file",
				},
			},
		},
		{
			Name:   "run",
			Usage:  "Run linter",
			Action: runAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "config-path",
					Usage:   "Load configuration from `PATH`",
					Aliases: []string{"c"},
				},
				&cli.StringFlag{
					Name:    "format",
					Usage:   "Output format. One of: [json, text]",
					Aliases: []string{"f"},
					Value:   "text",
				},
			},
		},
		{
			Name:   "version",
			Usage:  "Version of a tool",
			Action: versionAction,
		},
	}
}

func initAction(ctx *cli.Context) error {
	builtinConfig, err := loadBuiltinConfig()
	if err != nil {
		return err
	}

	workflowPaths := ctx.Args().Slice()
	if len(workflowPaths) == 0 {
		workflowPaths, err = gitHubWorkflows()
		if err != nil {
			return err
		}
	}

	if ctx.Bool("debug") {
		for _, p := range workflowPaths {
			fmt.Fprintf(os.Stderr, "workflow path: %s\n", p)
		}
	}

	config, err := buildConfig(workflowPaths)
	if err != nil {
		return err
	}
	config.merge(builtinConfig, true)

	path := ctx.String("output-path")
	if path == "" {
		path = defaultConfigPath
	}
	_, err = os.Stat(path)
	if err == nil && !ctx.Bool("overwrite") {
		return fmt.Errorf("already exists: %s", path)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := writeConfig(f, config); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Created %s\n", path)

	return nil
}

func runAction(ctx *cli.Context) error {
	config, err := lookupConfig(ctx.String("config-path"))
	if err != nil {
		return err
	}
	if ctx.Bool("debug") {
		fmt.Fprintf(os.Stderr, "config: %v\n", config)
	}

	workflows := ctx.Args().Slice()
	if len(workflows) == 0 {
		gitHubWorkflows, err := gitHubWorkflows()
		if err != nil {
			return err
		}
		workflows = gitHubWorkflows
	}

	var errs []LintErr
	for _, w := range workflows {
		linter := newLinter(config)
		if err := linter.Lint(w); err != nil {
			return err
		}
		errs = append(errs, linter.Errors()...)
	}

	if len(errs) > 0 {
		format := ctx.String("format")
		switch format {
		case "text":
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "%s\n", e)
			}
			os.Exit(1)
		case "json":
			if err := json.NewEncoder(os.Stdout).Encode(errs); err != nil {
				return err
			}

		default:
			return fmt.Errorf("unknown format: %s", format)
		}
	}

	return nil
}

func versionAction(ctx *cli.Context) error {
	fmt.Println(Version)
	return nil
}

const (
	gitHubWorkflowsGlob = ".github/workflows/*"
)

func gitHubWorkflows() ([]string, error) {
	matches, err := filepath.Glob(gitHubWorkflowsGlob)
	if err != nil {
		return nil, err
	}

	var workflows []string
	for _, m := range matches {
		if filepath.Ext(m) == ".yml" || filepath.Ext(m) == ".yaml" {
			workflows = append(workflows, m)
		}
	}

	return workflows, nil
}
