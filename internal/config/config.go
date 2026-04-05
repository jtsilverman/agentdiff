package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all agentdiff configuration.
type Config struct {
	Thresholds Thresholds `yaml:"thresholds"`
}

// Thresholds defines regression detection sensitivity.
type Thresholds struct {
	ToolScore float64 `yaml:"tool_score"`
	TextScore float64 `yaml:"text_score"`
	StepDelta int     `yaml:"step_delta"`
}

// DefaultConfig returns a Config with default threshold values.
func DefaultConfig() Config {
	return Config{
		Thresholds: Thresholds{
			ToolScore: 0.3,
			TextScore: 0.5,
			StepDelta: 5,
		},
	}
}

// Load reads .agentdiff.yaml from the given directory and merges with defaults.
// If the file does not exist, defaults are returned with no error.
func Load(dir string) (Config, error) {
	cfg := DefaultConfig()

	path := dir + "/.agentdiff.yaml"
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}

	// Parse into a temporary struct to detect which fields were set.
	var fileCfg fileConfig
	if err := yaml.Unmarshal(data, &fileCfg); err != nil {
		return cfg, err
	}

	// Merge: only override defaults for fields present in the file.
	if fileCfg.Thresholds != nil {
		if v, ok := fileCfg.Thresholds["tool_score"]; ok {
			cfg.Thresholds.ToolScore = toFloat64(v)
		}
		if v, ok := fileCfg.Thresholds["text_score"]; ok {
			cfg.Thresholds.TextScore = toFloat64(v)
		}
		if v, ok := fileCfg.Thresholds["step_delta"]; ok {
			cfg.Thresholds.StepDelta = toInt(v)
		}
	}

	return cfg, nil
}

// fileConfig is used for partial YAML parsing so we can detect which fields
// were explicitly set vs absent.
type fileConfig struct {
	Thresholds map[string]interface{} `yaml:"thresholds"`
}

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		return 0
	}
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}
