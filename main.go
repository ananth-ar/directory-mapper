package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Pattern struct {
	extension string // For patterns like "*.log"
	directory string // For patterns like "src/cmd/"
}

// PatternList represents an ordered list of patterns
type PatternList struct {
	patterns  []Pattern
	basePath  string
	matchType PatternType
}

// TreeNode represents a file or directory in the tree structure
type TreeNode struct {
	name     string
	isDir    bool
	children []*TreeNode
}

// PatternType indicates whether patterns are for ignoring or filtering
type PatternType int

const (
	Ignore PatternType = iota
	Filter
)

// determinePatternType checks which pattern file exists and should be used
func determinePatternType(ignoreFile, filterFile string) (string, PatternType, error) {
	ignoreExists := false
	filterExists := false

	if _, err := os.Stat(ignoreFile); err == nil {
		ignoreExists = true
	}
	if _, err := os.Stat(filterFile); err == nil {
		filterExists = true
	}

	// If both exist, use ignore file
	if ignoreExists {
		return ignoreFile, Ignore, nil
	}
	// If only filter exists, use filter file
	if filterExists {
		return filterFile, Filter, nil
	}
	// If neither exists, create and use ignore file
	if err := os.WriteFile(ignoreFile, []byte{}, 0644); err != nil {
		return "", Ignore, fmt.Errorf("error creating ignore file: %v", err)
	}
	return ignoreFile, Ignore, nil
}

// NewPatternList creates a new pattern list from a file
func NewPatternList(filename string, basePath string, matchType PatternType) (*PatternList, error) {
	pl := &PatternList{
		patterns:  make([]Pattern, 0),
		basePath:  basePath,
		matchType: matchType,
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %v", filename, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		pattern := strings.TrimSpace(scanner.Text())
		if pattern == "" || strings.HasPrefix(pattern, "#") {
			continue
		}
		if err := pl.AddPattern(pattern); err != nil {
			return nil, fmt.Errorf("error adding pattern %s: %v", pattern, err)
		}
	}

	return pl, scanner.Err()
}

// AddPattern adds a new pattern to the list
func (pl *PatternList) AddPattern(pattern string) error {
	p := Pattern{}

	// Handle file extension pattern (*.ext)
	if strings.HasPrefix(pattern, "*.") {
		p.extension = strings.TrimPrefix(pattern, "*")
	} else {
		// Handle directory pattern
		p.directory = filepath.Clean(pattern)
	}

	pl.patterns = append(pl.patterns, p)
	return nil
}

// Matches checks if a path matches any pattern in the list
func (pl *PatternList) Matches(path string) bool {
	if len(pl.patterns) == 0 {
		return pl.matchType == Filter // If no patterns and Filter mode, nothing matches
	}

	// Convert path to relative and clean
	relPath := path
	if filepath.IsAbs(path) {
		var err error
		relPath, err = filepath.Rel(pl.basePath, path)
		if err != nil {
			return false
		}
	}
	relPath = filepath.Clean(relPath)

	// Check each pattern
	for _, p := range pl.patterns {
		// Check file extension pattern
		if p.extension != "" && strings.HasSuffix(relPath, p.extension) {
			return true
		}

		// Check directory pattern
		if p.directory != "" {
			if strings.HasPrefix(relPath, p.directory) {
				return true
			}
		}
	}

	return false
}

// Common file patterns and directories to skip
var (
	skipDirs = map[string]bool{
		".git":         true,
		"node_modules": true,
		"bin":          true,
		"obj":          true,
		"build":        true,
		"dist":         true,
		"target":       true,
		".idea":        true,
		".vscode":      true,
		"__pycache__":  true,
		".next":        true,
		"vendor":       true,
	}

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

	skipFiles = map[string]bool{
		"project_structure.txt":     true,
		".project_structure_ignore": true,
		".project_structure_filter": true,
		".DS_Store":                 true,
		"Thumbs.db":                 true,
		".gitignore":                true,
		".env":                      true,
		".env.local":                true,
		"desktop.ini":               true,
	}

	maxFileSize = int64(50 * 1024 * 1024)
)

func shouldSkipFile(entry os.DirEntry, fullPath string, patterns *PatternList) (bool, error) {

	info, err := entry.Info()
	if err != nil {
		return false, fmt.Errorf("error getting file info: %v", err)
	}

	if patterns != nil {
		matches := patterns.Matches(fullPath)

		if patterns.matchType == Ignore {
			if matches {
				return true, nil
			}
		} else {
			if !matches {
				return true, nil
			}
		}
	}

	if skipFiles[entry.Name()] {
		return true, nil
	}

	if info.IsDir() && skipDirs[entry.Name()] {
		return true, nil
	}

	if !info.IsDir() {
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if skipExtensions[ext] {
			return true, nil
		}

		if info.Size() > maxFileSize {
			return true, nil
		}

		if err := checkReadPermission(fullPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Cannot read file %s: %v\n", fullPath, err)
			return true, nil
		}
	}

	return false, nil
}

func checkReadPermission(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

func createTree(root string, ignoreMatcher *PatternList) (*TreeNode, error) {
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

		skip, err := shouldSkipFile(entry, childPath, ignoreMatcher)
		if err != nil {
			return nil, fmt.Errorf("error checking file %s: %v", childPath, err)
		}
		if skip {
			continue
		}

		childNode, err := createTree(childPath, ignoreMatcher)
		if err != nil {
			return nil, err
		}
		rootNode.children = append(rootNode.children, childNode)
	}

	return rootNode, nil
}

func printTree(node *TreeNode, prefix string, isLast bool, output *os.File) {
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

	var displayName string
	if node.isDir {
		displayName = fmt.Sprintf("[%s]", node.name)
	} else {
		displayName = node.name
	}
	fmt.Fprintln(output, currentPrefix+displayName)

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

	for i, child := range node.children {
		isLastChild := i == len(node.children)-1
		printTree(child, childPrefix, isLastChild, output)
	}
}

func writeFileContents(node *TreeNode, currentPath string, output *os.File) error {
	fullPath := filepath.Join(currentPath, node.name)

	if !node.isDir {
		_, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("error checking file %s: %v", fullPath, err)
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not read file %s: %v\n", fullPath, err)
			return nil
		}

		fmt.Fprintf(output, "<%s>\n", node.name)
		fmt.Fprintf(output, "%s\n", string(content))
		fmt.Fprintf(output, "\n</%s>\n", node.name)
	}

	for _, child := range node.children {
		if err := writeFileContents(child, fullPath, output); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	ignoreFile := filepath.Join(currentDir, ".project_structure_ignore")
	filterFile := filepath.Join(currentDir, ".project_structure_filter")

	// Determine which pattern file to use
	patternFile, patternType, err := determinePatternType(ignoreFile, filterFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error determining pattern type: %v\n", err)
		os.Exit(1)
	}

	// Initialize pattern matcher
	patterns, err := NewPatternList(patternFile, currentDir, patternType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing patterns: %v\n", err)
		os.Exit(1)
	}

	outputFile, err := os.Create("project_structure.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	root, err := createTree(currentDir, patterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating tree structure: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(outputFile, "<Project_Structure>")
	printTree(root, "", true, outputFile)
	fmt.Fprintln(outputFile, "</Project_Structure>")

	err = writeFileContents(root, filepath.Dir(currentDir), outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file contents: %v\n", err)
		os.Exit(1)
	}

	patternTypeStr := "ignore"
	if patternType == Filter {
		patternTypeStr = "filter"
	}
	fmt.Printf("Project structure and file contents have been written to project_structure.txt using %s patterns\n", patternTypeStr)
}
