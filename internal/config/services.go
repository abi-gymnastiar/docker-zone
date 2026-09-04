package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dashboard/internal/domain"
	"gopkg.in/yaml.v3"
)

func LoadServices(dir string) (map[string]domain.ServiceConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read service configs: %w", err)
	}

	services := make(map[string]domain.ServiceConfig)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yml" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		var service domain.ServiceConfig
		if err := yaml.Unmarshal(data, &service); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		if service.Name == "" {
			service.Name = strings.TrimSuffix(entry.Name(), ".yml")
		}
		if service.Container == "" {
			service.Container = service.Name
		}
		services[service.Name] = service
	}
	return services, nil
}
