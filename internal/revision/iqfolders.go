package revision

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Agent-Field/codeaf/internal/store"
)

// iqFolderReads answers an action point that names folders by what the run read under them (IQ). A folder is a
// word that ends in a slash -- "shop/", "src/api/" -- so prose is never mistaken for a path. Every folder the
// point names must have a read under it; the reason counts them, "the run read 3 files under shop/, 1 under tests/".
func iqFolderReads(point store.AcceptancePoint, read []string) string {
	if point.Kind != "action" && point.Kind != "reply" || len(read) == 0 {
		return ""
	}
	folders := folderWords(point.Behaviour + " " + point.Quote)
	if len(folders) == 0 {
		return ""
	}
	counts := make([]string, 0, len(folders))
	for index, folder := range folders {
		under := 0
		for _, path := range read {
			if strings.Contains("/"+filepath.ToSlash(path), "/"+folder) {
				under++
			}
		}
		if under == 0 {
			return ""
		}
		files := "files"
		if under == 1 {
			files = "file"
		}
		if index == 0 {
			counts = append(counts, fmt.Sprintf("%d %s under %s", under, files, folder))
		} else {
			counts = append(counts, fmt.Sprintf("%d under %s", under, folder))
		}
	}
	return "the run read " + strings.Join(counts, ", ")
}

// folderWords reads slash-ended words, normalised to "a/b/", in the order the point names them.
func folderWords(text string) []string {
	var folders []string
	for _, field := range strings.Fields(text) {
		word := strings.Trim(field, "`'\"()[],;:")
		if !strings.HasSuffix(word, "/") {
			continue
		}
		word = strings.TrimLeft(strings.TrimPrefix(word, "./"), "/")
		if strings.Trim(word, "/") == "" || strings.Contains(word, "..") || !hasASCIILetter(word) {
			continue
		}
		duplicate := false
		for _, kept := range folders {
			if strings.EqualFold(kept, word) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			folders = append(folders, word)
		}
	}
	return folders
}
