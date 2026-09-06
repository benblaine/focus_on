// Package config resolves where the focuson data directory lives, persisting
// the choice locally so it only has to be set once.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DataDir string `toml:"data_dir"`
}

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "focuson", "config.toml"), nil
}

// Load reads the config file. It returns (Config{}, false, nil) if no config
// file exists yet — that's the normal first-run state, not an error.
func Load() (Config, bool, error) {
	p, err := path()
	if err != nil {
		return Config{}, false, err
	}
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return Config{}, false, nil
	}
	var cfg Config
	if _, err := toml.DecodeFile(p, &cfg); err != nil {
		return Config{}, false, err
	}
	return cfg, true, nil
}

// Save writes the config file, creating its parent directory if needed.
func Save(cfg Config) error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

// DefaultDataDir is offered as a starting suggestion on first run.
func DefaultDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "focuson-data"), nil
}
