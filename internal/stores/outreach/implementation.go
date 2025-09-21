package outreach

import (
	"fmt"

	"github.com/ethanbaker/assistant/pkg/outreach"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Store handles storage and retrieval of outreach implementations using MySQL
type Store struct {
	db *gorm.DB
}

// NewStore creates a new outreach store with MySQL connection
func NewStore(databaseURL string) (*Store, error) {
	db, err := gorm.Open(mysql.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}

	// Auto-migrate tables
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	return store, nil
}

// migrate creates or updates the required database tables
func (s *Store) migrate() error {
	return s.db.AutoMigrate(&ImplementationModel{})
}

// SaveImplementation stores an implementation by client ID
func (s *Store) SaveImplementation(impl *outreach.Implementation) error {
	if impl.ClientID == "" {
		return fmt.Errorf("client_id cannot be empty")
	}
	if impl.CallbackURL == "" {
		return fmt.Errorf("callback_url cannot be empty")
	}

	model := &ImplementationModel{
		ClientID:     impl.ClientID,
		CallbackURL:  impl.CallbackURL,
		ClientSecret: impl.ClientSecret,
		Active:       impl.Active,
	}

	// Check if implementation already exists
	var existing ImplementationModel
	result := s.db.Where("client_id = ?", impl.ClientID).First(&existing)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Create new record
			if err := s.db.Create(model).Error; err != nil {
				return fmt.Errorf("failed to create implementation: %w", err)
			}
		} else {
			return fmt.Errorf("failed to check existing implementation: %w", result.Error)
		}
	} else {
		// Update existing record
		if err := s.db.Model(&existing).Updates(map[string]any{
			"callback_url":  impl.CallbackURL,
			"client_secret": impl.ClientSecret,
			"active":        impl.Active,
		}).Error; err != nil {
			return fmt.Errorf("failed to update implementation: %w", err)
		}
	}

	return nil
}

// GetImplementation retrieves an implementation by client ID
func (s *Store) GetImplementation(clientID string) (*outreach.Implementation, error) {
	if clientID == "" {
		return nil, fmt.Errorf("client_id cannot be empty")
	}

	var model ImplementationModel
	result := s.db.Where("client_id = ?", clientID).First(&model)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("implementation with client_id '%s' not found", clientID)
		}
		return nil, fmt.Errorf("failed to get implementation: %w", result.Error)
	}

	impl := &outreach.Implementation{
		ClientID:     model.ClientID,
		CallbackURL:  model.CallbackURL,
		ClientSecret: model.ClientSecret,
		Active:       model.Active,
	}

	return impl, nil
}

// DisableImplementation removes an implementation by client ID
func (s *Store) DisableImplementation(clientID string) error {
	if clientID == "" {
		return fmt.Errorf("client_id cannot be empty")
	}

	result := s.db.Model(&ImplementationModel{}).Where("client_id = ?", clientID).Update("active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to delete implementation: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("implementation with client_id '%s' not found", clientID)
	}

	return nil
}

// ListImplementations returns all registered implementations
func (s *Store) ListImplementations() []*outreach.Implementation {
	var models []ImplementationModel
	if err := s.db.Order("client_id").Find(&models).Where("active = ?", true).Error; err != nil {
		// Return empty slice on error rather than nil to maintain interface contract
		return []*outreach.Implementation{}
	}

	implementations := make([]*outreach.Implementation, len(models))
	for i, model := range models {
		implementations[i] = &outreach.Implementation{
			ClientID:     model.ClientID,
			CallbackURL:  model.CallbackURL,
			ClientSecret: model.ClientSecret,
			Active:       model.Active,
		}
	}

	return implementations
}

// Exists checks if an implementation with the given client ID exists
func (s *Store) Exists(clientID string) bool {
	if clientID == "" {
		return false
	}

	var count int64
	s.db.Model(&ImplementationModel{}).Where("client_id = ?", clientID).Where("active = ?", true).Count(&count)
	return count > 0
}

// Close closes the database connection
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm.DB: %w", err)
	}
	return sqlDB.Close()
}

// AuthenticateImplementation validates client ID and secret combination
func (s *Store) AuthenticateImplementation(clientID, clientSecret string) (*outreach.Implementation, error) {
	if clientID == "" {
		return nil, fmt.Errorf("client_id cannot be empty")
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("client_secret cannot be empty")
	}

	var model ImplementationModel
	result := s.db.Where("client_id = ? AND client_secret = ?", clientID, clientSecret).First(&model)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid client credentials")
		}
		return nil, fmt.Errorf("failed to authenticate implementation: %w", result.Error)
	}

	return &outreach.Implementation{
		ClientID:     model.ClientID,
		CallbackURL:  model.CallbackURL,
		ClientSecret: model.ClientSecret,
		Active:       model.Active,
	}, nil
}
