package domain

type ServiceConfig struct {
	Name        string   `yaml:"name" json:"name"`
	Container   string   `yaml:"container" json:"container"`
	Description string   `yaml:"description" json:"description"`
	Actions     []string `yaml:"actions" json:"actions"`
}

type Service struct {
	ServiceConfig
	ContainerID string `json:"containerId"`
	Status      string `json:"status"`
	Running     bool   `json:"running"`
}
