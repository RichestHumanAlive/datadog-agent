package ebpf

import (
	"embed"
	"io/ioutil"
	"os"
	"path/filepath"
)

//go:embed headers
var headersFS embed.FS

// loadHeadersToTmpfs takes Go DIs needed header files from the embeded filesystem
// and loads them into the systems tmpfs so clang can find them
//
// The returned string is the directory path of the headers directory in tmpfs
func loadHeadersToTmpfs(directory string) (string, error) {
	fs, err := headersFS.ReadDir("headers")
	if err != nil {
		return "", err
	}

	tmpHeaderDir, err := os.MkdirTemp(directory, "dd-di-bpf-headers")
	if err != nil {
		return "", err
	}

	for _, entry := range fs {
		content, err := headersFS.ReadFile(filepath.Join("headers", entry.Name()))
		if err != nil {
			return "", err
		}
		err = ioutil.WriteFile(filepath.Join(tmpHeaderDir, entry.Name()), content, 0644)
		if err != nil {
			return "", err
		}
	}
	return tmpHeaderDir, nil
}
