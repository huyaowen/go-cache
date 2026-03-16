package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/coderiser/go-cache/pkg/scan"
)

var (
	force   = flag.Bool("force", false, "force full scan (ignore incremental)")
	verbose = flag.Bool("v", false, "verbose output")
	help    = flag.Bool("h", false, "show help")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gocache scan [options] [directories]\n\n")
		fmt.Fprintf(os.Stderr, "Scan Go files for cache annotations and generate wrappers.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gocache scan ./...                    # scan all packages\n")
		fmt.Fprintf(os.Stderr, "  gocache scan ./service                # scan specific directory\n")
		fmt.Fprintf(os.Stderr, "  gocache scan -force ./...             # force full scan\n")
		fmt.Fprintf(os.Stderr, "  gocache scan -v ./service ./repo      # scan multiple directories with verbose\n")
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"."}
	}

	var dirs []string
	for _, arg := range args {
		if strings.HasSuffix(arg, "/...") {
			dir := strings.TrimSuffix(arg, "/...")
			dirs = append(dirs, dir)
		} else {
			dirs = append(dirs, arg)
		}
	}

	config := &scan.Config{
		Dirs:    dirs,
		Force:   *force,
		Verbose: *verbose,
	}

	fmt.Println("🔍 Scanning for cache annotations...")

	scanner := scan.NewScanner(config)
	result, err := scanner.Scan(dirs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Scan error: %v\n", err)
		os.Exit(1)
	}

	if len(result.Packages) == 0 {
		fmt.Println("⚠️  No cache annotations found")
		return
	}

	totalServices := 0
	totalMethods := 0
	for _, pkg := range result.Packages {
		for _, file := range pkg.Files {
			totalServices += len(file.Services)
			for _, svc := range file.Services {
				totalMethods += len(svc.Methods)
			}
		}
	}

	if *verbose {
		fmt.Printf("📦 Found %d packages, %d services, %d annotated methods\n\n",
			len(result.Packages), totalServices, totalMethods)
	}

	generator := scan.NewGenerator(config)
	if err := generator.Generate(result); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Generate error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Generated cache wrappers successfully!")
}
