package game

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var codeExtensions = map[string]bool{
	".go":    true,
	".js":    true,
	".ts":    true,
	".tsx":   true,
	".jsx":   true,
	".py":    true,
	".rb":    true,
	".rs":    true,
	".c":     true,
	".cpp":   true,
	".cc":    true,
	".h":     true,
	".hpp":   true,
	".java":  true,
	".cs":    true,
	".swift": true,
	".kt":    true,
	".scala": true,
	".php":   true,
	".pl":    true,
	".sh":    true,
	".bash":  true,
	".zsh":   true,
	".lua":   true,
	".r":     true,
	".m":     true,
	".mm":    true,
	".zig":   true,
	".nim":   true,
	".ex":    true,
	".exs":   true,
	".erl":   true,
	".hs":    true,
	".ml":    true,
	".fs":    true,
	".clj":   true,
	".lisp":  true,
	".el":    true,
	".vim":   true,
}

type CodeFile struct {
	Path    string
	Lines   []string
	SHA     string
}

func findCodeFiles(root string, minLines, maxFiles int) ([]CodeFile, error) {
	var candidates []CodeFile

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden directories and common non-code directories
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check extension
		ext := strings.ToLower(filepath.Ext(path))
		if !codeExtensions[ext] {
			return nil
		}

		// Read and count lines
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		var lines []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if len(lines) >= minLines {
			// Compute SHA256 of file content
			content := strings.Join(lines, "\n")
			hash := sha256.Sum256([]byte(content))
			sha := string(hash[:])

			candidates = append(candidates, CodeFile{
				Path:  path,
				Lines: lines,
				SHA:   sha,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by line count (prefer longer files for more interesting backgrounds)
	sort.Slice(candidates, func(i, j int) bool {
		return len(candidates[i].Lines) > len(candidates[j].Lines)
	})

	// Take up to maxFiles
	if len(candidates) > maxFiles {
		candidates = candidates[:maxFiles]
	}

	return candidates, nil
}

func computeSeed(files []CodeFile) int64 {
	h := sha256.New()
	for _, f := range files {
		h.Write([]byte(f.SHA))
	}
	sum := h.Sum(nil)
	return int64(binary.BigEndian.Uint64(sum[:8]))
}
