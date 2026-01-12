package config

import (
	"github.com/urfave/cli/v2"
	"github.com/zihaofu245/chaturbate-dvr/entity"
)

// New initializes a new Config struct with values from the CLI context.
func New(c *cli.Context) (*entity.Config, error) {
	cfg := &entity.Config{
		Version:       c.App.Version,
		Username:      c.String("username"),
		AdminUsername: c.String("admin-username"),
		AdminPassword: c.String("admin-password"),
		Framerate:     c.Int("framerate"),
		Resolution:    c.Int("resolution"),
		Pattern:       c.String("pattern"),
		MaxDuration:   c.Int("max-duration"),
		MaxFilesize:   c.Int("max-filesize"),
		Port:          c.String("port"),
		Interval:      c.Int("interval"),
		Cookies:       c.String("cookies"),
		UserAgent:     c.String("user-agent"),
		Domain:        c.String("domain"),
	}
	if err := Load(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
