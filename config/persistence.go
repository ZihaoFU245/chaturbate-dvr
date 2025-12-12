package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/teacat/chaturbate-dvr/entity"
)

const configPath = "./conf/config.json"

type persistentConfig struct {
	Cookies   string `json:"cookies"`
	UserAgent string `json:"user_agent"`
}

// Save persists the UserAgent and Cookies to a JSON file.
func Save(cfg *entity.Config) error {
	pCfg := persistentConfig{
		Cookies:   cfg.Cookies,
		UserAgent: cfg.UserAgent,
	}
	b, err := json.MarshalIndent(pCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0777); err != nil {
		return fmt.Errorf("mkdir all: %w", err)
	}
	if err := os.WriteFile(configPath, b, 0777); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// Load reads the UserAgent and Cookies from the JSON file and updates the config
// if the fields are empty.
func Load(cfg *entity.Config) error {
	b, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	var pCfg persistentConfig
	if err := json.Unmarshal(b, &pCfg); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	if cfg.Cookies == "" && pCfg.Cookies != "" {
		cfg.Cookies = pCfg.Cookies
	}
	if cfg.UserAgent == "" && pCfg.UserAgent != "" {
		cfg.UserAgent = pCfg.UserAgent
	}
	return nil
}
