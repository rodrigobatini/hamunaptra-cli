package localproj

import (
	"fmt"
	"os"
	"strings"
)

const FileName = "hamunaptra.yaml"

func ReadID(dir string) (string, error) {
	b, err := os.ReadFile(dir + "/" + FileName)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "project_id:") {
			id := strings.TrimSpace(strings.TrimPrefix(line, "project_id:"))
			id = strings.Trim(id, `"'`)
			if id != "" {
				return id, nil
			}
		}
	}
	return "", fmt.Errorf("project_id not found in %s", FileName)
}

func WriteID(dir, id string) error {
	content := "# Hamunaptra — linked cloud project\nproject_id: " + id + "\n"
	return os.WriteFile(dir+"/"+FileName, []byte(content), 0o644)
}
