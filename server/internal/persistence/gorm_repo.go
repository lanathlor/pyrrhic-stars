package persistence

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// GormRepo implements Repository using GORM.
type GormRepo struct {
	db *gorm.DB
}

// NewGormRepo opens a database connection and runs migrations.
// driver is "sqlite" or "postgres". For sqlite the dsn is ignored (in-memory).
func NewGormRepo(driver, dsn string) (*GormRepo, error) {
	var dialector gorm.Dialector
	switch driver {
	case "postgres":
		if dsn == "" {
			return nil, errors.New("persistence: postgres requires POSTGRES_DSN")
		}
		dialector = postgres.Open(dsn)
	default:
		dialector = sqlite.Open("file::memory:?cache=shared")
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\n", log.LstdFlags),
			logger.Config{
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
			},
		),
	})
	if err != nil {
		return nil, fmt.Errorf("persistence open: %w", err)
	}

	if err := db.AutoMigrate(&Player{}, &Character{}); err != nil {
		return nil, fmt.Errorf("persistence migrate: %w", err)
	}

	// Drop legacy unique index if it exists (allows multiple chars per class).
	migrator := db.Migrator()
	if migrator.HasIndex(&Character{}, "idx_player_class") {
		_ = migrator.DropIndex(&Character{}, "idx_player_class")
	}
	// Drop legacy unique index on player username if it exists.
	if migrator.HasIndex(&Player{}, "idx_players_username") {
		_ = migrator.DropIndex(&Player{}, "idx_players_username")
	}

	// Backfill empty character names for pre-existing records.
	db.Exec("UPDATE characters SET name = 'Char_' || id WHERE name = '' OR name IS NULL")

	return &GormRepo{db: db}, nil
}

func (r *GormRepo) UpsertPlayer(id, username string) error {
	player := Player{ID: id, Username: username}
	result := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&player)
	return result.Error
}

func (r *GormRepo) GetPlayer(id string) (*Player, error) {
	var player Player
	result := r.db.First(&player, "id = ?", id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &player, result.Error
}

func (r *GormRepo) CreateCharacter(c *Character) error {
	return r.db.Create(c).Error
}

func (r *GormRepo) UpdateCharacterPosition(charID uint, posX, posY, posZ, rotY float64) error {
	return r.db.Model(&Character{}).Where("id = ?", charID).Updates(map[string]any{
		"pos_x": posX, "pos_y": posY, "pos_z": posZ, "rot_y": rotY,
	}).Error
}

func (r *GormRepo) GetCharacterByID(id uint) (*Character, error) {
	var c Character
	result := r.db.First(&c, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, result.Error
}

func (r *GormRepo) GetCharacters(playerID string) ([]*Character, error) {
	var chars []*Character
	result := r.db.Where("player_id = ?", playerID).Order("updated_at DESC").Find(&chars)
	return chars, result.Error
}

func (r *GormRepo) IsCharacterNameTaken(name string) (bool, error) {
	var count int64
	result := r.db.Model(&Character{}).Where("name = ?", name).Count(&count)
	return count > 0, result.Error
}

func (r *GormRepo) CountCharacters(playerID string) (int64, error) {
	var count int64
	result := r.db.Model(&Character{}).Where("player_id = ?", playerID).Count(&count)
	return count, result.Error
}
