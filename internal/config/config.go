package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Web struct {
	Title           string `yaml:"title" json:"title"`
	Icon            string `yaml:"icon" json:"icon"`
	Header          string `yaml:"header" json:"header"`
	Subheader       string `yaml:"subheader" json:"subheader"`
	Footer          string `yaml:"footer" json:"footer"`
	BackgroundImage string `yaml:"backgroundImage,omitempty" json:"backgroundImage,omitempty"`
	BackgroundScale string `yaml:"backgroundScale,omitempty" json:"backgroundScale,omitempty"`
}

type Evil struct {
	Images []string `yaml:"images" json:"images"`
}

type Config struct {
	Web  Web   `yaml:"web" json:"web"`
	Evil *Evil `yaml:"evil,omitempty" json:"evil,omitempty"`
}

type Manager struct {
	path    string
	mu      sync.RWMutex
	config  Config
	modTime time.Time
}

func New(path string) (*Manager, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		initial := defaults()
		data, marshalErr := yaml.Marshal(initial)
		if marshalErr != nil {
			return nil, marshalErr
		}
		data = append(data, []byte(`
# Optional evil overlay. Uncomment and edit to show the EVIL BUTTON.
# evil:
#   images:
#     - https://example.com/evil.gif
#     - /data/evil/local-image.png
`)...)
		if err := os.WriteFile(path, data, 0640); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else if err := appendTemplateIfMissing(path); err != nil {
		return nil, err
	}
	manager := &Manager{path: path}
	if err := manager.reload(); err != nil {
		return nil, err
	}
	return manager, nil
}

func appendTemplateIfMissing(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.Contains(string(data), "# Optional evil overlay.") {
		return nil
	}
	data = append(data, []byte(`
# Optional evil overlay. Uncomment and edit to show the EVIL BUTTON.
# evil:
#   images:
#     - https://example.com/evil.gif
#     - /data/evil/local-image.png
`)...)
	return os.WriteFile(path, data, 0640)
}

func (m *Manager) Current() (Config, error) {
	info, err := os.Stat(m.path)
	if err != nil {
		return Config{}, err
	}
	m.mu.RLock()
	changed := info.ModTime().After(m.modTime)
	current := m.config
	m.mu.RUnlock()
	if changed {
		if err := m.reload(); err != nil {
			return Config{}, err
		}
		m.mu.RLock()
		current = m.config
		m.mu.RUnlock()
	}
	return current, nil
}

func (m *Manager) reload() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}
	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		return err
	}
	loaded = normalize(loaded)
	info, err := os.Stat(m.path)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.config = loaded
	m.modTime = info.ModTime()
	m.mu.Unlock()
	return nil
}

func defaults() Config {
	value := Config{
		Web: Web{
			Title:           "My Docker Zone",
			Icon:            "https://cdn.jsdelivr.net/gh/selfhst/icons/svg/linux.svg",
			Header:          "★ MY DOCKER ZONE ★",
			Subheader:       "tiny control panel / very serious technology",
			Footer:          "docker zone, Developed by Jimi - with love <3",
			BackgroundScale: "tile",
		},
	}
	if env := os.Getenv("WEB_TITLE"); env != "" {
		value.Web.Title = env
	}
	if env := os.Getenv("WEB_ICON"); env != "" {
		value.Web.Icon = env
	}
	if env := os.Getenv("WEB_HEADER"); env != "" {
		value.Web.Header = env
	}
	if env := os.Getenv("WEB_SUBHEADER"); env != "" {
		value.Web.Subheader = env
	}
	if env := os.Getenv("WEB_FOOTER"); env != "" {
		value.Web.Footer = env
	}
	if env := os.Getenv("DEFAULT_BACKGROUND_IMAGE"); env != "" {
		value.Web.BackgroundImage = env
	}
	if env := os.Getenv("DEFAULT_BACKGROUND_SCALE"); env != "" {
		value.Web.BackgroundScale = env
	}
	return value
}

func normalize(value Config) Config {
	fallback := defaults()
	if strings.TrimSpace(value.Web.Title) == "" {
		value.Web.Title = fallback.Web.Title
	}
	if strings.TrimSpace(value.Web.Header) == "" {
		value.Web.Header = fallback.Web.Header
	}
	if strings.TrimSpace(value.Web.Subheader) == "" {
		value.Web.Subheader = fallback.Web.Subheader
	}
	if strings.TrimSpace(value.Web.Footer) == "" {
		value.Web.Footer = fallback.Web.Footer
	}
	if strings.TrimSpace(value.Web.BackgroundImage) == "" {
		value.Web.BackgroundImage = fallback.Web.BackgroundImage
	}
	if value.Web.BackgroundScale != "tile" && value.Web.BackgroundScale != "stretch" && value.Web.BackgroundScale != "zoom" {
		value.Web.BackgroundScale = fallback.Web.BackgroundScale
	}
	return value
}
