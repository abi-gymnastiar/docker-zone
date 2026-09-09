package domain

type ServiceConfig struct {
	Name        string   `yaml:"name" json:"name"`
	Container   string   `yaml:"container" json:"container"`
	ContainerID string   `json:"containerId"`
	Description string   `yaml:"description" json:"description"`
	Groups      []string `yaml:"groups" json:"groups"`
	Actions     []string `yaml:"actions" json:"actions"`
	Enabled     bool     `json:"enabled"`
	Orphaned    bool     `json:"orphaned"`
}

type Service struct {
	ServiceConfig
	ContainerID string `json:"containerId"`
	Status      string `json:"status"`
	Running     bool   `json:"running"`
}

type DiscoveredService struct {
	ContainerID string
	Container   string
	Name        string
	Description string
	Actions     []string
	Groups      []string
	Running     bool
	Status      string
}
