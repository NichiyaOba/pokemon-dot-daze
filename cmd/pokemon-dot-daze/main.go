// Command pokemon-dot-daze renders a Pokémon sprite that fills in as the
// owner's GitHub contributions accumulate.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/NichiyaOba/pokemon-dot-daze/progress"
	"github.com/NichiyaOba/pokemon-dot-daze/render"
	"github.com/NichiyaOba/pokemon-dot-daze/source"
	"github.com/NichiyaOba/pokemon-dot-daze/sprite"
)

type config struct {
	user    string
	token   string
	dex     int
	out     string
	cell    int
	gap     int
	theme   string
	caption bool
	commits int
	// commitsSet records whether --commits was passed, so that 0 is a usable
	// value and a negative one is still rejected.
	commitsSet bool
}

func main() {
	err := run(os.Args[1:])
	if err == nil {
		return
	}
	// flag already printed the usage text; asking for help is not a failure.
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	fmt.Fprintln(os.Stderr, "pokemon-dot-daze:", err)
	os.Exit(1)
}

func run(args []string) error {
	cfg, err := parseFlags(args)
	if err != nil {
		return err
	}

	s, err := sprite.ByDex(cfg.dex)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	commits, err := resolveCommits(ctx, cfg)
	if err != nil {
		return err
	}

	p, err := progress.Compute(s, commits)
	if err != nil {
		return err
	}

	opts, err := renderOptions(cfg)
	if err != nil {
		return err
	}

	svg, err := render.SVG(p, opts)
	if err != nil {
		return err
	}

	if err := writeFile(cfg.out, svg); err != nil {
		return err
	}

	reportProgress(p, cfg.out)
	return nil
}

func parseFlags(args []string) (config, error) {
	fs := flag.NewFlagSet("pokemon-dot-daze", flag.ContinueOnError)

	cfg := config{}
	fs.StringVar(&cfg.user, "user", os.Getenv("GITHUB_REPOSITORY_OWNER"), "GitHub login to read contributions for")
	fs.StringVar(&cfg.token, "token", os.Getenv("GITHUB_TOKEN"), "GitHub API token")
	fs.IntVar(&cfg.dex, "dex", 1, "Pokédex number of the sprite to render")
	fs.StringVar(&cfg.out, "out", filepath.Join("assets", "pokemon.svg"), "path to write the SVG to")
	fs.IntVar(&cfg.cell, "cell", render.DefaultCellSize, "edge length of one dot, in pixels")
	fs.IntVar(&cfg.gap, "gap", render.DefaultGap, "space between dots, in pixels")
	fs.StringVar(&cfg.theme, "theme", string(render.ThemeAuto), "colour scheme: auto, light or dark")
	fs.BoolVar(&cfg.caption, "caption", true, "draw the name and progress beneath the artwork")
	fs.IntVar(&cfg.commits, "commits", 0, "use this contribution count instead of calling the API (for testing)")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: pokemon-dot-daze [flags]\n\n"+
			"Renders a Pokémon sprite that reveals one dot per %d GitHub contributions.\n\n"+
			"Flags:\n", progress.CommitsPerDot)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	fs.Visit(func(f *flag.Flag) {
		if f.Name == "commits" {
			cfg.commitsSet = true
		}
	})
	if cfg.commitsSet && cfg.commits < 0 {
		return config{}, fmt.Errorf("--commits must not be negative, got %d", cfg.commits)
	}
	if cfg.out == "" {
		return config{}, errors.New("--out must not be empty")
	}
	return cfg, nil
}

// resolveCommits prefers an explicit --commits over an API call, which keeps
// local runs and tests offline.
func resolveCommits(ctx context.Context, cfg config) (int, error) {
	if cfg.commitsSet {
		return cfg.commits, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	src := source.GitHub{Login: cfg.user, Token: cfg.token}
	commits, err := src.TotalContributions(ctx)
	if err != nil {
		return 0, fmt.Errorf("reading contributions: %w", err)
	}
	return commits, nil
}

func renderOptions(cfg config) (render.Options, error) {
	opts := render.DefaultOptions()
	opts.CellSize = cfg.cell
	opts.Gap = cfg.gap
	opts.Theme = render.Theme(cfg.theme)
	opts.ShowCaption = cfg.caption

	if err := opts.Validate(); err != nil {
		return render.Options{}, err
	}
	return opts, nil
}

func writeFile(path string, data []byte) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func reportProgress(p progress.Progress, out string) {
	fmt.Printf("#%03d %s: %d/%d dots (%d%%) from %d contributions -> %s\n",
		p.Sprite.Dex, p.Sprite.Name, p.Revealed, p.Total, p.Percent(), p.Commits, out)

	if p.IsComplete() {
		fmt.Printf("%s is complete!\n", p.Sprite.Name)
		return
	}
	fmt.Printf("%d more contributions to complete %s.\n", p.CommitsRemaining(), p.Sprite.Name)
}
