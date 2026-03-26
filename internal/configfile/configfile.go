package configfile

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const dirName = "hamunaptra"
const fileName = "config.json"

type Config struct {
	APIBase     string `json:"api_base"`
	AccessToken string `json:"access_token,omitempty"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", dirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

func Load() (*Config, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{APIBase: os.Getenv("HAMUNAPTRA_API")}, nil
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.APIBase == "" {
		c.APIBase = os.Getenv("HAMUNAPTRA_API")
	}
	return &c, nil
}

func Save(c *Config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600)
}

func DefaultAPIBase() string {
	if v := os.Getenv("HAMUNAPTRA_API"); v != "" {
		return v
	}
	return "http://127.0.0.1:8081"
}
