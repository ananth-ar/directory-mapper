package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// createIndent returns a string of spaces based on the depth
func createIndent(depth int) string {
	return strings.Repeat("  ", depth + 1) // +1 to account for initial indent
}

func main() {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Create output file
	outputFile, err := os.Create("project_structure.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	// Write header to file
	fmt.Fprintln(outputFile, "--- Project Structure ---")

	// Walk through directory
	err = filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(currentDir, path)
		if err != nil {
			return err
		}

		// Skip the output file itself
		if relPath == "project_structure.txt" {
			return nil
		}

		// Calculate indent level based on path depth
		var indent string
		if relPath == "." {
			indent = ""
		} else {
			depth := len(strings.Split(filepath.Dir(relPath), string(os.PathSeparator))) - 1
			if depth < 0 {
				depth = 0
			}
			indent = createIndent(depth)
		}

		// Format the line differently for directories
		if info.IsDir() {
			fmt.Fprintf(outputFile, "%s[%s]\n", indent, filepath.Base(path))
		} else {
			fmt.Fprintf(outputFile, "%s%s\n", indent, filepath.Base(path))
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Project structure has been written to project_structure.txt")
}