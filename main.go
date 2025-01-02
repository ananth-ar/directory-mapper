package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Pattern represents a single pattern with its properties
type Pattern struct {
	original  string   // Original pattern string
	segments  []string // Pattern broken into path segments
	isNegation bool   // Whether this is a negation pattern
	isExact   bool    // Whether this is an exact match pattern
	isDir     bool    // Whether this specifically matches directories
}

// PatternList represents an ordered list of patterns with matching logic
type PatternList struct {
	patterns []*Pattern
	basePath string // Base path for relative pattern matching
}

// TreeNode represents a file or directory in the tree structure
type TreeNode struct {
	name     string
	isDir    bool
	children []*TreeNode
}

func (p Pattern) String() string {
    return fmt.Sprintf("{original:%s, isNegation:%v, isDir:%v}", p.original, p.isNegation, p.isDir)
}

// NewPatternList creates a new pattern list from a file
func NewPatternList(filename, basePath string) (*PatternList, error) {
	pl := &PatternList{
		patterns: make([]*Pattern, 0),
		basePath: basePath,
	}

	// Create file if it doesn't exist
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		if err := os.WriteFile(filename, []byte{}, 0644); err != nil {
			return nil, fmt.Errorf("error creating file %s: %v", filename, err)
		}
		return pl, nil
	}

	// Read patterns from file
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
    // fmt.Printf("%+v\n", pl.patterns) 
	return pl, scanner.Err()
}

// AddPattern adds a new pattern to the list
func (pl *PatternList) AddPattern(pattern string) error {
	p := &Pattern{
		original:   pattern,
		isNegation: strings.HasPrefix(pattern, "!"),
		isDir:      strings.HasSuffix(pattern, "/"),
	}

	// Handle negation
	if p.isNegation {
		pattern = pattern[1:]
	}

	// Handle directory suffix
	if p.isDir {
		pattern = pattern[:len(pattern)-1]
	}

	// Clean and split pattern
	pattern = filepath.Clean(pattern)
	// if filepath.IsAbs(pattern) {
	// 	rel, err := filepath.Rel(pl.basePath, pattern)
	// 	if err != nil {
	// 		return fmt.Errorf("error converting absolute path to relative: %v", err)
	// 	}
	// 	pattern = rel
	// }

	// Convert pattern to segments
	p.segments = splitPattern(pattern) // hr\fgf\**.txt => [hr fgf **.txt]
	p.isExact = !strings.Contains(pattern, "*") || strings.Contains(pattern, "?")

	pl.patterns = append(pl.patterns, p)
	return nil
}

// splitPattern splits a pattern into segments handling wildcards
func splitPattern(pattern string) []string {
	// Special case for double asterisk
	pattern = strings.ReplaceAll(pattern, "**", "\x00")
	
	// Split on path separator
	segments := strings.Split(filepath.ToSlash(pattern), "/")
	
	// Restore double asterisks
	for i, seg := range segments {
		segments[i] = strings.ReplaceAll(seg, "\x00", "**")
	}
	
	return segments
}

// Matches checks if a path matches any pattern in the list
func (pl *PatternList) Matches(path string) bool {
	if len(pl.patterns) == 0 {
		return false
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
	relPath = filepath.ToSlash(filepath.Clean(relPath))
    // now=> temp/gg.go
	
	// Get path info
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	isDir := info.IsDir()

	// Check each pattern in order
	result := false
	for _, p := range pl.patterns { 
		if p.matches(relPath, isDir) {// to be changed
			result = !p.isNegation
		}
	}
	return result
}

// matches checks if a path matches a single pattern
func (p *Pattern) matches(path string, isDir bool) bool {
	// Directory-specific patterns only match directories
	if p.isDir && !isDir {
		return false
	}

	// Split path into segments
	pathSegments := strings.Split(path, "/")

	// For exact matches, compare directly
	if p.isExact {
		return strings.Join(p.segments, "/") == path
	}

	return p.matchSegments(p.segments, pathSegments)
}

// matchSegments performs recursive segment matching
func (p *Pattern) matchSegments(patternSegs, pathSegs []string) bool {
	// Base cases
	if len(patternSegs) == 0 {
		return len(pathSegs) == 0
	}
	if len(pathSegs) == 0 {
		// Remaining pattern segments must all be "**"
		for _, seg := range patternSegs {
			if seg != "**" {
				return false
			}
		}
		return true
	}

	// Current segments to match
	patSeg := patternSegs[0]
	pathSeg := pathSegs[0]

	// Handle double asterisk
	if patSeg == "**" {
		// Try matching with and without consuming path segment
		return p.matchSegments(patternSegs[1:], pathSegs) ||
			p.matchSegments(patternSegs, pathSegs[1:])
	}

	// Handle single asterisk and question mark
	if matchWildcard(patSeg, pathSeg) {
		return p.matchSegments(patternSegs[1:], pathSegs[1:])
	}

	return false
}

// matchWildcard matches a single pattern segment against a path segment
func matchWildcard(pattern, path string) bool {
	// Convert pattern to regex
	regex := strings.Builder{}
	regex.WriteString("^")
	
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			regex.WriteString(".*")
		case '?':
			regex.WriteString(".")
		default:
			// Escape regex special characters
			if strings.ContainsRune("[](){}+^$|\\.", rune(pattern[i])) {
				regex.WriteRune('\\')
			}
			regex.WriteByte(pattern[i])
		}
	}
	
	regex.WriteString("$")
	
	// Use strings.Contains for simple cases
	if pattern == "*" {
		return true
	}
	
	// Use prefix/suffix for simple wildcards
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		middle := pattern[1:len(pattern)-1]
		return strings.Contains(path, middle)
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(path, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(path, pattern[:len(pattern)-1])
	}
	
	return pattern == path
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
		"project_structure.txt":           true,
		".project_structure_ignore":       true,
		".project_structure_filter":       true,
		".DS_Store":                      true,
		"Thumbs.db":                      true,
		".gitignore":                     true,
		".env":                           true,
		".env.local":                     true,
		"desktop.ini":                    true,
	}

	maxFileSize = int64(50 * 1024 * 1024)
)

func shouldSkipFile(entry os.DirEntry, fullPath string, ignoreMatcher *PatternList) (bool, error) {
	info, err := entry.Info()
	if err != nil {
		return false, fmt.Errorf("error getting file info: %v", err)
	}

	// Check ignore patterns first
	if ignoreMatcher != nil && ignoreMatcher.Matches(fullPath) {
		return true, nil
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
		// if filterMatcher != nil && !filterMatcher.Matches(fullPath) {
		// 	return nil
		// }

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
		fmt.Fprintf(output, "%s", string(content))
		fmt.Fprintf(output, "\n</%s>\n", node.name)
	}
	
	for _, child := range node.children {
		err := writeFileContents(child, fullPath, output)
		if err != nil {
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

	ignoreMatcher, err := NewPatternList(".project_structure_ignore", currentDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing ignore patterns: %v\n", err)
		os.Exit(1)
	}

	// filterMatcher, err := NewPatternList(".project_structure_filter", currentDir)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error initializing filter patterns: %v\n", err)
	// 	os.Exit(1)
	// }

	outputFile, err := os.Create("project_structure.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	root, err := createTree(currentDir, ignoreMatcher)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating tree structure: %v\n", err)
		os.Exit(1)
	}
    fmt.Fprintln(outputFile, "<Project_Structure>")
	printTree(root, "", true, outputFile)
	fmt.Fprintln(outputFile, "</Project_Structure>")


	// Only use filterMatcher if there are actual filter patterns
	// var activeFilterMatcher *PatternList
	// if len(filterMatcher.patterns) > 0 {
	// 	activeFilterMatcher = filterMatcher
	// }
	fmt.Fprintln(outputFile, "</File_Contents>")
	err = writeFileContents(root, filepath.Dir(currentDir), outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file contents: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(outputFile, "\n</File_Contents>")

	fmt.Println("Project structure and file contents have been written to project_structure.txt")
}