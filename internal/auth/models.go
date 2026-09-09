package auth

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type Session struct {
	User  User
	Token string
}

type AdminUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type AdminGroup struct {
	ID      int64         `json:"id"`
	Name    string        `json:"name"`
	Members []GroupMember `json:"members"`
}

type GroupMember struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type ServiceGroups struct {
	Name        string   `json:"name"`
	Container   string   `json:"container"`
	ContainerID string   `json:"containerId"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
	Groups      []string `json:"groups"`
	Enabled     bool     `json:"enabled"`
	Orphaned    bool     `json:"orphaned"`
}
