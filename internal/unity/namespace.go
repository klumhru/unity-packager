package unity

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var namespaceRegex = regexp.MustCompile(`^\s*namespace\s+([\w.]+)`)

// InferRootNamespace scans .cs files in a directory tree and returns the most common
// top-level namespace. For example, if files declare "Google.Protobuf", "Google.Protobuf.Collections",
// and "Google.Protobuf.Reflection", this returns "Google.Protobuf".
func InferRootNamespace(dir string) string {
	counts := make(map[string]int)

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".cs") {
			return nil
		}

		ns := extractNamespace(path)
		if ns != "" {
			counts[ns]++
		}
		return nil
	})

	if len(counts) == 0 {
		return ""
	}

	// Find the most common top-level namespace prefix
	// First, collect all namespaces
	var namespaces []string
	for ns := range counts {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)

	// Find shortest common prefix among all namespaces
	// But weighted by frequency — pick the prefix that covers the most files
	prefixCounts := make(map[string]int)
	for ns, count := range counts {
		parts := strings.Split(ns, ".")
		// Build progressively longer prefixes
		for i := 1; i <= len(parts); i++ {
			prefix := strings.Join(parts[:i], ".")
			prefixCounts[prefix] += count
		}
	}

	// Find the shortest prefix that covers the majority of files
	totalFiles := 0
	for _, c := range counts {
		totalFiles += c
	}

	// Sort prefixes by depth (shorter first), then by coverage
	type prefixInfo struct {
		prefix string
		count  int
		depth  int
	}
	var prefixes []prefixInfo
	for p, c := range prefixCounts {
		prefixes = append(prefixes, prefixInfo{
			prefix: p,
			count:  c,
			depth:  strings.Count(p, ".") + 1,
		})
	}
	sort.Slice(prefixes, func(i, j int) bool {
		// Prefer deeper namespaces that still cover most files
		if prefixes[i].count == prefixes[j].count {
			return prefixes[i].depth > prefixes[j].depth
		}
		return prefixes[i].count > prefixes[j].count
	})

	if len(prefixes) > 0 {
		// Return the deepest prefix that still covers >50% of files
		for _, p := range prefixes {
			if p.count >= totalFiles/2 && p.depth >= 2 {
				return p.prefix
			}
		}
		return prefixes[0].prefix
	}

	return ""
}

func extractNamespace(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		matches := namespaceRegex.FindStringSubmatch(scanner.Text())
		if len(matches) >= 2 {
			return matches[1]
		}
	}
	return ""
}
