package linta

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed config.builtin.yml
var builtinConfig string

type Config struct {
	Repositories map[string]*PermissionConfig `yaml:"repositories"`
	Ignores      map[string]*IgnoreConfig     `yaml:"ignores"`
}

func (c *Config) String() string {
	var b strings.Builder

	for k, v := range c.Repositories {
		b.WriteString(fmt.Sprintf("[%s]\n", k))
		b.WriteString(fmt.Sprintf("- %s", v))
	}

	for k, v := range c.Ignores {
		b.WriteString(fmt.Sprintf("[%s]\n", k))
		for j, w := range *v {
			b.WriteString(fmt.Sprintf("- %s\n", j))
			for _, ww := range w {
				b.WriteString(fmt.Sprintf("  - %s\n", ww))
			}
		}
	}

	return b.String()
}

func (c *Config) ignoreEnabled(path, job, perm string) bool {
	if c.Ignores == nil {
		return false
	}

	cc, ok := c.Ignores[path]
	if !ok {
		return false
	}

	jc, ok := (*cc)[job]
	if !ok {
		return false
	}

	for _, p := range jc {
		if p == perm {
			return true
		}
	}

	return false
}

type PermissionConfig map[string]string

func (c *PermissionConfig) String() string {
	var b strings.Builder

	for k, v := range *c {
		b.WriteString(fmt.Sprintf("  %s:%s\n", k, v))
	}

	return b.String()
}

type IgnoreConfig map[string][]string

func newConfig() *Config {
	return &Config{
		Repositories: make(map[string]*PermissionConfig),
	}
}

func (c *Config) updateConfig(repository string, pc *PermissionConfig) {
	c.Repositories[repository] = pc
}

func (c *Config) getPermissionConfig(r string) *PermissionConfig {
	return c.Repositories[r]
}

func (c *Config) merge(cc *Config, overwrite bool) {
	for k, v := range cc.Repositories {
		if _, ok := c.Repositories[k]; ok && !overwrite {
			continue
		}

		c.Repositories[k] = v
	}
}

const (
	defaultConfigPath = ".linta.yml"
)

func lookupConfig(path string) (*Config, error) {
	if path != "" {
		return loadConfigPath(path)
	}

	_, err := os.Stat(defaultConfigPath)
	if err == nil {
		return loadConfigPath(defaultConfigPath)
	}

	return loadBuiltinConfig()
}

func loadConfigPath(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return loadConfig(f)
}

func loadBuiltinConfig() (*Config, error) {
	r := strings.NewReader(builtinConfig)
	return loadConfig(r)
}

func loadConfig(r io.Reader) (*Config, error) {
	var c Config

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	if err := c.validate(); err != nil {
		return nil, err
	}

	return &c, nil
}

func writeConfig(w io.Writer, config *Config) error {
	e := yaml.NewEncoder(w)
	e.SetIndent(2)
	return e.Encode(config)
}

func (c Config) validate() error {
	for r, pc := range c.Repositories {
		if len(strings.Split(r, "/")) < 2 {
			return fmt.Errorf("invalid repository name: %s", r)
		}

		if err := pc.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c PermissionConfig) validate() error {
	for k, v := range c {
		if _, ok := allPermissionScopes[k]; !ok {
			return fmt.Errorf("invalid permission scope: %s", k)
		}

		switch v {
		case noneScope:
		case readScope:
		case writeScope:
		default:
			return fmt.Errorf("invalid permission value: %s", v)
		}
	}

	return nil
}

const (
	noneScope  = "none"
	readScope  = "read"
	writeScope = "write"
)

var allPermissionScopes = map[string]struct{}{
	"actions":             {},
	"checks":              {},
	"contents":            {},
	"deployments":         {},
	"id-token":            {},
	"issues":              {},
	"discussions":         {},
	"packages":            {},
	"pages":               {},
	"pull-requests":       {},
	"repository-projects": {},
	"security-events":     {},
	"statuses":            {},
}
