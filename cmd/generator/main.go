package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"generator/parser"
	"generator/extractor"
	gen "generator/generator"
)

var (
	outputDir string
	verbose   bool
)

func init() {
	flag.StringVar(&outputDir, "output", "./generated", "Output directory for generated code")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <source-files...>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Go Cache Framework Code Generator\n")
		fmt.Fprintf(os.Stderr, "Parses Go source files with @cacheable annotations and generates wrapper code.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s --output ./pkg/cache/wrapper ./services/*.go\n", os.Args[0])
	}
}

func logVerbose(format string, args ...interface{}) {
	if verbose {
		log.Printf(format, args...)
	}
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No source files specified\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	logVerbose("Output directory: %s", outputDir)

	// Initialize parser
	p := parser.NewParser()

	// Parse all source files
	var allServices []*extractor.ServiceInfo

	for _, srcFile := range args {
		logVerbose("Processing file: %s", srcFile)

		// Parse the file
		astFile, err := p.ParseFile(srcFile)
		if err != nil {
			log.Printf("Warning: Failed to parse %s: %v", srcFile, err)
			continue
		}

		// Extract annotations
		annotations := p.ExtractAnnotations(astFile)
		logVerbose("Found %d annotations in %s", len(annotations), srcFile)

		// Extract service info
		services := p.ExtractServices(astFile, annotations)
		logVerbose("Found %d services in %s", len(services), srcFile)

		allServices = append(allServices, services...)
	}

	if len(allServices) == 0 {
		log.Println("No services with @cacheable annotations found")
		os.Exit(0)
	}

	// Initialize generator
	g := gen.NewGenerator()

	// Generate wrapper code
	logVerbose("Generating wrapper code...")
	wrapperCode, err := g.GenerateWrapper(allServices)
	if err != nil {
		log.Fatalf("Failed to generate wrapper code: %v", err)
	}

	// Write wrapper file
	wrapperPath := filepath.Join(outputDir, "wrapper.go")
	if err := os.WriteFile(wrapperPath, wrapperCode, 0644); err != nil {
		log.Fatalf("Failed to write wrapper file: %v", err)
	}
	logVerbose("Written: %s", wrapperPath)

	// Generate registry code
	logVerbose("Generating registry code...")
	registryCode, err := g.GenerateRegistry(allServices)
	if err != nil {
		log.Fatalf("Failed to generate registry code: %v", err)
	}

	// Write registry file
	registryPath := filepath.Join(outputDir, "registry.go")
	if err := os.WriteFile(registryPath, registryCode, 0644); err != nil {
		log.Fatalf("Failed to write registry file: %v", err)
	}
	logVerbose("Written: %s", registryPath)

	fmt.Printf("✓ Generated %d service wrappers\n", len(allServices))
	fmt.Printf("✓ Output directory: %s\n", outputDir)
}
