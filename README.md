# Project Structure Generator

A command-line tool written in Go that generates a comprehensive documentation of your project's structure and file contents. This tool is inspired by the VSCode Project Structure extension but operates as a standalone application.

## Features

- Generates a detailed tree view of your project's directory structure
- Includes file contents in the output
- Supports custom ignore patterns through `.project_structure_ignore` file
- Automatically skips common binary files, build artifacts, and large files
- Handles file permission issues gracefully
- XML-formatted output for easy parsing

## Installation

```bash
# Clone the repository
git clone https://github.com/ananth-ar/directory-mapper.git
cd project-structure-generator

# Build the project
go build
```

## Usage

1. Navigate to your project directory
2. Create a `.project_structure_ignore` file (optional) to specify patterns to ignore
3. Run the generator:

```bash
./project-structure-generator
```

The tool will create a `project_structure.txt` file containing:
- A tree view of your project structure
- The contents of all non-ignored files

### Ignore Patterns

Create a `.project_structure_ignore` file in your project root to specify patterns to ignore:

```
# Ignore specific files or directories
node_modules
dist/temp/

```

## Default Exclusions

The tool automatically excludes:
- Common build directories (bin, obj, dist, build)
- Development directories (.git, node_modules, vendor)
- Binary and large files
- System files (.DS_Store, Thumbs.db)
- Files larger than 50MB

## Output Format

The generated `project_structure.txt` file uses a simple XML-like format:

```
<Project_Structure>
[root]
├── [src]
│   ├── main.go
│   └── utils.go
└── README.md
</Project_Structure>

<File_Contents>
<main.go>
// File contents here
</main.go>
</File_Contents>
```
