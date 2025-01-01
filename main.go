package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TreeNode represents a file or directory in the tree structure
type TreeNode struct {
	name     string
	isDir    bool
	children []*TreeNode
}

// createTree creates a tree structure starting from the given root path
// Common file patterns and directories to skip
var (
	// Directories to skip
	skipDirs = map[string]bool{
		".git":           true,
		"node_modules":   true,
		"bin":           true,
		"obj":           true,
		"build":         true,
		"dist":          true,
		"target":        true,
		".idea":         true,
		".vscode":       true,
		"__pycache__":   true,
		".next":         true,
		"vendor":        true,
	}

	// File extensions to skip (binary, system, and special files)
	skipExtensions = map[string]bool{
		".exe":    true,
		".dll":    true,
		".so":     true,
		".dylib":  true,
		".bin":    true,
		".obj":    true,
		".class":  true,
		".pyc":    true,
		".pdb":    true,
		".cache":  true,
		".jpg":    true,
		".jpeg":   true,
		".png":    true,
		".gif":    true,
		".ico":    true,
		".pdf":    true,
		".zip":    true,
		".tar":    true,
		".gz":     true,
		".rar":    true,
		".7z":     true,
		".db":     true,
		".sqlite": true,
		".mdb":    true,
		".iso":    true,
		".img":    true,
		".log":    true,
		".lock":   true,
	}

	// Special filenames to skip
	skipFiles = map[string]bool{
		"project_structure.txt": true,
		".DS_Store":            true,
		"Thumbs.db":            true,
		".gitignore":           true,
		".env":                 true,
		".env.local":           true,
		"desktop.ini":          true,
	}

	// Size limit for text files (50MB)
	maxFileSize = int64(50 * 1024 * 1024)
)

// shouldSkipFile determines if a file or directory should be skipped
func shouldSkipFile(entry os.DirEntry, fullPath string) (bool, error) {
	// Get file info
	info, err := entry.Info()
	if err != nil {
		return false, fmt.Errorf("error getting file info: %v", err)
	}

	// Skip if it's in skipFiles
	if skipFiles[entry.Name()] {
		return true, nil
	}

	// Skip if it's a directory in skipDirs
	if info.IsDir() && skipDirs[entry.Name()] {
		return true, nil
	}

	// Skip if it's not a directory and has a skipped extension
	if !info.IsDir() {
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if skipExtensions[ext] {
			return true, nil
		}

		// Skip large files
		if info.Size() > maxFileSize {
			return true, nil
		}

		// Skip files without read permission
		if err := checkReadPermission(fullPath); err != nil {
			return true, nil
		}
	}

	return false, nil
}

// checkReadPermission checks if the file can be read
func checkReadPermission(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

func createTree(root string) (*TreeNode, error) {
	rootInfo, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("error getting root info: %v", err)
	}

	rootNode := &TreeNode{
		name:     rootInfo.Name(),
		isDir:    rootInfo.IsDir(),
		children: make([]*TreeNode, 0),
	}

	if !rootInfo.IsDir() {
		return rootNode, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %v", err)
	}

	for _, entry := range entries {
		childPath := filepath.Join(root, entry.Name())
		
		// Check if file should be skipped
		skip, err := shouldSkipFile(entry, childPath)
		if err != nil {
			return nil, fmt.Errorf("error checking file %s: %v", childPath, err)
		}
		if skip {
			continue
		}

		// childPath := filepath.Join(root, entry.Name())
		childNode, err := createTree(childPath)
		if err != nil {
			return nil, err
		}
		rootNode.children = append(rootNode.children, childNode)
	}

	return rootNode, nil
}

// printTree prints the tree structure with proper indentation and branch characters
func printTree(node *TreeNode, prefix string, isLast bool, output *os.File) {
	// Create the current line's prefix
	var currentPrefix string
	if prefix == "" {
		currentPrefix = ""
	} else {
		if isLast {
			currentPrefix = prefix + "└── "
		} else {
			currentPrefix = prefix + "├── "
		}
	}

	// Print the current node
	var displayName string
	if node.isDir {
		displayName = fmt.Sprintf("[%s]", node.name)
	} else {
		displayName = node.name
	}
	fmt.Fprintln(output, currentPrefix+displayName)

	// Prepare the prefix for children
	var childPrefix string
	if prefix == "" {
		childPrefix = "    "
	} else {
		if isLast {
			childPrefix = prefix + "    "
		} else {
			childPrefix = prefix + "│   "
		}
	}

	// Print children
	for i, child := range node.children {
		isLastChild := i == len(node.children)-1
		printTree(child, childPrefix, isLastChild, output)
	}
}

// writeFileContents writes the contents of all files in the tree
func writeFileContents(node *TreeNode, currentPath string, output *os.File) error {
    fullPath := filepath.Join(currentPath, node.name)
    
    if !node.isDir {
        // First check if file still exists
        _, err := os.Stat(fullPath)
        if err != nil {
            if os.IsNotExist(err) {
                return nil
            }
            return fmt.Errorf("error checking file %s: %v", fullPath, err)
        }

        // Read and write file contents
        content, err := os.ReadFile(fullPath)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Warning: Could not read file %s: %v\n", fullPath, err)
            return nil
        }
        
        fmt.Fprintf(output, "\n\n--- File: %s ---\n", fullPath)
        fmt.Fprintf(output, "%s", string(content))
    }
    
    // Recursively process children, passing the current full path
    for _, child := range node.children {
        err := writeFileContents(child, fullPath, output)
        if err != nil {
            return err
        }
    }
    
    return nil
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

	// Create and print the tree
	root, err := createTree(currentDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating tree structure: %v\n", err)
		os.Exit(1)
	}

	printTree(root, "", true, outputFile)
	
	// Write file contents section
	fmt.Fprintln(outputFile, "\n--- File Contents ---")
	err = writeFileContents(root, filepath.Dir(currentDir), outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file contents: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Project structure and file contents have been written to project_structure.txt")
}