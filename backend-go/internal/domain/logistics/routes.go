package logistics

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DefaultEngine is the rate engine loaded from carrier-rates YAML files
// during RegisterRoutes. It is exposed so other modules (e.g. the sourcing
// toolregistry) can wire the same engine into their profit calculations
// without re-loading the YAML. It may be nil if no rate files were found.
var DefaultEngine *RateEngine

// defaultCarrierRatesDir is the relative path used when no override is provided.
// It resolves correctly when the server is launched from the backend-go/
// working directory (e.g. `go run cmd/server/main.go` or the docker WORKDIR).
const defaultCarrierRatesDir = "./carrier-rates"

// envCarrierRatesDir is the environment variable used to override the
// carrier-rates directory (e.g. for docker volume mounts or CI).
const envCarrierRatesDir = "CARRIER_RATES_DIR"

// RegisterRoutes registers logistics routes on the given router group.
// Carrier rate tables are loaded from the carrier-rates directory
// (overridable via CARRIER_RATES_DIR env var). Missing directory or
// parse failures are logged as warnings but do not crash the server.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	tables := loadCarrierRateTables(logger)
	svc := NewService(tables)
	DefaultEngine = svc.engine
	h := NewHandler(svc, logger)

	group := rg.Group("/logistics")
	{
		group.POST("/quote", h.GetQuotes)
	}
}

// loadCarrierRateTables reads all *.yaml / *.yml files from the carrier-rates
// directory and returns the concatenated rate table entries. Errors for
// individual files are logged as warnings; a missing directory is also a warn.
func loadCarrierRateTables(logger *zap.Logger) []RateTableEntry {
	dir := strings.TrimSpace(os.Getenv(envCarrierRatesDir))
	if dir == "" {
		dir = defaultCarrierRatesDir
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn("logistics: carrier-rates directory not found, falling back to empty rate table",
				zap.String("dir", dir),
				zap.String("hint", "set "+envCarrierRatesDir+" env var or create the directory"),
			)
			return nil
		}
		logger.Warn("logistics: stat carrier-rates directory failed, falling back to empty rate table",
			zap.String("dir", dir),
			zap.Error(err),
		)
		return nil
	}
	if !info.IsDir() {
		logger.Warn("logistics: carrier-rates path is not a directory, falling back to empty rate table",
			zap.String("path", dir),
		)
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		logger.Warn("logistics: read carrier-rates directory failed, falling back to empty rate table",
			zap.String("dir", dir),
			zap.Error(err),
		)
		return nil
	}

	// Sort entries by name for deterministic load order across platforms.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var all []RateTableEntry
	loadedFiles := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Warn("logistics: skip carrier-rate file (read failed)",
				zap.String("file", path),
				zap.Error(err),
			)
			continue
		}
		table, err := LoadRateTableFromYAML(data)
		if err != nil {
			logger.Warn("logistics: skip carrier-rate file (parse failed)",
				zap.String("file", path),
				zap.Error(err),
			)
			continue
		}
		all = append(all, table...)
		loadedFiles++
	}

	logger.Info("logistics: loaded carrier rate tables",
		zap.String("dir", dir),
		zap.Int("files", loadedFiles),
		zap.Int("entries", len(all)),
	)
	return all
}
