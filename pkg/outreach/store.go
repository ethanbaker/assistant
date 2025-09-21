package outreach

// Implementation represents a registered outreach implementation
type Implementation struct {
	ClientID     string `json:"client_id"`
	CallbackURL  string `json:"callback_url"`
	ClientSecret string `json:"client_secret"`
	Active       bool   `json:"active"`
}

// Store defines the interface for outreach storage operations
type StoreInterface interface {
	SaveImplementation(impl *Implementation) error
	GetImplementation(clientID string) (*Implementation, error)
	DisableImplementation(clientID string) error
	ListImplementations() []*Implementation
	Exists(clientID string) bool
	AuthenticateImplementation(clientID, clientSecret string) (*Implementation, error)
}
