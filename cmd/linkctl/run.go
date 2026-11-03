package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"os"
	"strings"

	"github.com/legalt/rd-linkforge/internal/config"
	"github.com/legalt/rd-linkforge/internal/link"
	"github.com/legalt/rd-linkforge/internal/linkio"
	"github.com/legalt/rd-linkforge/internal/store"
)

func run(ctx context.Context, args []string, env func(string) (string, bool), stdout, stderr io.Writer) int {
	if env == nil {
		env = os.LookupEnv
	}
	levelText, ok := env("LF_LOG_LEVEL")
	if !ok || strings.TrimSpace(levelText) == "" {
		levelText = "info"
	}
	level, err := config.ParseLevel(levelText)
	if err != nil {
		fmt.Fprintf(stderr, "LF_LOG_LEVEL: %v\n", err)
		return 2
	}
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level}))
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: linkctl {import|export|config|version} [flags]")
		return 2
	}

	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	var input, database, output string
	switch args[0] {
	case "import":
		flags.StringVar(&input, "in", "", "target file (required)")
		flags.StringVar(&database, "db", "", "JSON Lines database (required)")
	case "export":
		flags.StringVar(&database, "db", "", "JSON Lines database (required)")
		flags.StringVar(&output, "out", "", "CSV output file (default stdout)")
	case "config", "version":
	default:
		fmt.Fprintln(stderr, "unknown command; usage: linkctl {import|export|config|version} [flags]")
		return 2
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || (args[0] == "import" && (input == "" || database == "")) ||
		(args[0] == "export" && database == "") {
		fmt.Fprintln(stderr, "invalid arguments: supply the required flags and no positional arguments")
		flags.Usage()
		return 2
	}
	if err := ctx.Err(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	switch args[0] {
	case "import":
		var skipped int
		skipped, err = importLinks(ctx, input, database, stdout, logger)
		if err == nil && skipped > 0 {
			return 1
		}
	case "export":
		err = exportLinks(ctx, database, output, stdout)
	case "config":
		var cfg config.Config
		cfg, err = config.NewEnvLoader(env).Load(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "invalid configuration: %v\n", err)
			return 1
		}
		logger.InfoContext(ctx, "configuration loaded", "config", cfg)
		_, err = fmt.Fprintln(stdout, "config ok")
	case "version":
		_, err = fmt.Fprintf(stdout, "linkctl %s\n", version)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func importLinks(ctx context.Context, input, database string, stdout io.Writer, logger *slog.Logger) (int, error) {
	imported, skipped := 0, 0
	logger.InfoContext(ctx, "import started")
	defer func() {
		logger.InfoContext(ctx, "import finished", "imported", imported, "skipped", skipped)
	}()

	targets, err := loadTargets(ctx, input)
	if err != nil {
		return 0, err
	}
	existing, err := linkio.LoadFile(database)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return 0, err
	}
	st := store.New()
	var maxID uint64
	for _, l := range existing {
		if _, err := st.Create(ctx, l); err != nil {
			return 0, fmt.Errorf("load database %q: %w", database, err)
		}
		maxID = max(maxID, l.ID)
	}
	if maxID == math.MaxUint64 {
		return 0, errors.New("database ID range exhausted")
	}
	svc := link.NewService(st, link.WithFirstID(maxID+1))
	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			return skipped, err
		}
		l, err := svc.Shorten(ctx, link.ShortenRequest{Target: target.URL, OwnerID: "cli"})
		if ctxErr := ctx.Err(); ctxErr != nil {
			return skipped, ctxErr
		}
		if err != nil {
			skipped++
			logger.WarnContext(ctx, "target skipped", "line", target.Line, "error", err)
			continue
		}
		imported++
		logger.DebugContext(ctx, "target imported", "line", target.Line, "id", l.ID)
	}
	// Do not open or truncate the database if cancellation interrupted the import.
	if err := ctx.Err(); err != nil {
		return skipped, err
	}
	if err := linkio.SaveFile(database, st.All()); err != nil {
		return skipped, err
	}
	_, err = fmt.Fprintf(stdout, "imported: %d\nskipped: %d\n", imported, skipped)
	return skipped, err
}

func loadTargets(ctx context.Context, path string) ([]linkio.Target, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open targets %q: %w", path, err)
	}
	targets, readErr := linkio.ReadTargets(contextReader{ctx: ctx, reader: file})
	if err := errors.Join(readErr, file.Close()); err != nil {
		return nil, fmt.Errorf("read targets %q: %w", path, err)
	}
	return targets, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func exportLinks(ctx context.Context, database, output string, stdout io.Writer) error {
	links, err := linkio.LoadFile(database)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if output == "" {
		return linkio.WriteCSV(stdout, links)
	}
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create CSV %q: %w", output, err)
	}
	if err := errors.Join(linkio.WriteCSV(file, links), file.Close()); err != nil {
		return fmt.Errorf("write CSV %q: %w", output, err)
	}
	return nil
}
