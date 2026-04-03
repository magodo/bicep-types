package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/bicep-types/src/bicep-types-go/index"
	"github.com/Azure/bicep-types/src/bicep-types-go/types"
)

func main() {
	typeFiles := []index.TypeFile{}

	var (
		dir        = flag.String("dir", "", "Input types JSON file")
		outputFile = flag.String("output", "", "Output file (defaults to stdout)")
	)

	flag.Parse()

	if *dir == "" {
		printUsage()
		return
	}

	var output *os.File
	if *outputFile == "" {
		output = os.Stdout
	} else {
		output, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			if closeErr := output.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Error closing output file: %v\n", closeErr)
			}
		}()
	}

	if err := filepath.Walk(*dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Base(path) != "types.json" {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read types file %s: %w", path, err)
		}

		var raw []json.RawMessage
		if err := json.Unmarshal(b, &raw); err != nil {
			return fmt.Errorf("failed to unmarshal types array %s: %w", path, err)
		}

		ts := make([]types.Type, len(raw))
		for i, r := range raw {
			typ, err := types.UnmarshalType(r)
			if err != nil {
				return fmt.Errorf("failed to unmarshal type at index %d in %s: %w", i, path, err)
			}
			ts[i] = typ
		}
		relPath, _ := filepath.Rel(*dir, path)
		typeFiles = append(typeFiles, index.TypeFile{
			RelativePath: relPath,
			Types:        ts,
		})

		return nil
	}); err != nil {
		log.Fatal(err)
	}

	idx := index.BuildIndex(typeFiles, func(s string) { log.Println(s) }, nil, nil)

	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		log.Fatalf("marshaling the index: %v", err)
	}

	n, err := output.Write(b)
	if err != nil {
		log.Fatalf("writing output: %v", err)
	}
	if n != len(b) {
		log.Fatal("not all output is written")
	}
}

func printUsage() {
	fmt.Println("bicep-index - Build the bicep-types-go index.json by processing types files.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  bicep-index -dir <dir> [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -dir <dir>        The directory contains the types files (required)")
	fmt.Println("  -output <file>    Output file (default: stdout)")
	fmt.Println()
}
