package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all agentdiff configuration.
type Config struct {
	Thresholds Thresholds     `yaml:"thresholds"`
	CI         CIConfig       `yaml:"ci"`
	Baseline   BaselineConfig `yaml:"baseline"`
}

// CIConfig holds CI/CD integration settings.
type CIConfig struct {
	BaselinePath     string `yaml:"baseline_path"`
	FailOnStyleDrift bool   `yaml:"fail_on_style_drift"`
}

// BaselineConfig holds statistical baseline settings.
type BaselineConfig struct {
	Runs       int     `yaml:"runs"`
	Confidence float64 `yaml:"confidence"`
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
		CI: CIConfig{
			BaselinePath:     "",
			FailOnStyleDrift: false,
		},
		Baseline: BaselineConfig{
			Runs:       5,
			Confidence: 0.95,
		},
	}
}

// Load reads .agentdiff.yaml from the given directory and merges with defaults.
// If the file does not exist, defaults are returned with no error.
func Load(dir string) (Config, error) {
	cfg := DefaultConfig()

	path := filepath.Join(dir, ".agentdiff.yaml")
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

	if fileCfg.CI != nil {
		if v, ok := fileCfg.CI["baseline_path"]; ok {
			cfg.CI.BaselinePath = toString(v)
		}
		if v, ok := fileCfg.CI["fail_on_style_drift"]; ok {
			cfg.CI.FailOnStyleDrift = toBool(v)
		}
	}

	if fileCfg.Baseline != nil {
		if v, ok := fileCfg.Baseline["runs"]; ok {
			cfg.Baseline.Runs = toInt(v)
		}
		if v, ok := fileCfg.Baseline["confidence"]; ok {
			cfg.Baseline.Confidence = toFloat64(v)
		}
	}

	// Validate confidence is in (0.0, 1.0).
	if cfg.Baseline.Confidence <= 0.0 || cfg.Baseline.Confidence >= 1.0 {
		return cfg, fmt.Errorf("baseline.confidence must be in (0.0, 1.0), got %v", cfg.Baseline.Confidence)
	}

	return cfg, nil
}

// fileConfig is used for partial YAML parsing so we can detect which fields
// were explicitly set vs absent.
type fileConfig struct {
	Thresholds map[string]interface{} `yaml:"thresholds"`
	CI         map[string]interface{} `yaml:"ci"`
	Baseline   map[string]interface{} `yaml:"baseline"`
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

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
