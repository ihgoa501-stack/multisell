package sdk

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lingmirror/backend-go/internal/aios/runtime"
	"go.uber.org/zap"
)

// SetupAllAgents reads all YAML agent definitions from the agents/ directory
// and registers each one with the provided runtime. Returns the list of
// successfully registered agent IDs and the first error encountered (if any).
//
// The agents directory is resolved relative to this source file's location.
func SetupAllAgents(rt *runtime.Runtime, logger *zap.Logger) ([]string, error) {
	dir := filepath.Join("agents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Try from the package directory if running from a different cwd.
		// This handles both `go run` and `go test` scenarios.
		dir = "internal/aios/sdk/agents"
		entries, err = os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("read agents directory: %w", err)
		}
	}

	// Collect and sort YAML files for deterministic registration order.
	var yamlFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			yamlFiles = append(yamlFiles, filepath.Join(dir, name))
		}
	}
	sort.Strings(yamlFiles)

	if len(yamlFiles) == 0 {
		logger.Warn("no YAML agent definition files found", zap.String("directory", dir))
		return nil, nil
	}

	var registered []string
	var firstErr error

	for _, f := range yamlFiles {
		if err := RegisterFromYAML(rt, f); err != nil {
			logger.Error("failed to register agent from YAML",
				zap.String("file", f),
				zap.Error(err),
			)
			if firstErr == nil {
				firstErr = fmt.Errorf("register from %s: %w", filepath.Base(f), err)
			}
			continue
		}
		registered = append(registered, filepath.Base(f))
		logger.Info("agent registered from YAML",
			zap.String("file", filepath.Base(f)),
		)
	}

	return registered, firstErr
}
