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

	if err := db.AutoMigrate(&User{}, &Character{}, &CharacterItem{}, &CharacterEquipment{}, &CharacterLoadout{}, &CharacterFluxCommitment{}); err != nil {
		return nil, fmt.Errorf("persistence migrate: %w", err)
	}

	// Drop legacy unique index if it exists (allows multiple chars per class).
	migrator := db.Migrator()
	if migrator.HasIndex(&Character{}, "idx_player_class") {
		_ = migrator.DropIndex(&Character{}, "idx_player_class")
	}
	// Drop legacy unique index on player username if it exists.
	if migrator.HasIndex(&User{}, "idx_players_username") {
		_ = migrator.DropIndex(&User{}, "idx_players_username")
	}

	// Backfill empty character names for pre-existing records.
	db.Exec("UPDATE characters SET name = 'Char_' || id WHERE name = '' OR name IS NULL")

	return &GormRepo{db: db}, nil
}

func (r *GormRepo) UpsertUser(id, username string) error {
	u := User{ID: id, Username: username}
	result := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&u)
	return result.Error
}

func (r *GormRepo) GetUser(id string) (*User, error) {
	var u User
	result := r.db.First(&u, "id = ?", id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, result.Error
}

func (r *GormRepo) CreateCharacter(c *Character) error {
	return r.db.Create(c).Error
}

func (r *GormRepo) UpdateCharacterPosition(charID uint, posX, posY, posZ, rotY float64) error {
	return r.db.Model(&Character{}).Where("id = ?", charID).Updates(map[string]any{
		"pos_x": posX, "pos_y": posY, "pos_z": posZ, "rot_y": rotY,
	}).Error
}

func (r *GormRepo) UpdateCharacterSpec(charID uint, specID string) error {
	return r.db.Model(&Character{}).Where("id = ?", charID).Update("spec_id", specID).Error
}

func (r *GormRepo) GetCharacterByID(id uint) (*Character, error) {
	var c Character
	result := r.db.First(&c, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, result.Error
}

func (r *GormRepo) GetCharacters(userID string) ([]*Character, error) {
	var chars []*Character
	result := r.db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&chars)
	return chars, result.Error
}

func (r *GormRepo) IsCharacterNameTaken(name string) (bool, error) {
	var count int64
	result := r.db.Model(&Character{}).Where("name = ?", name).Count(&count)
	return count > 0, result.Error
}

func (r *GormRepo) CountCharacters(userID string) (int64, error) {
	var count int64
	result := r.db.Model(&Character{}).Where("user_id = ?", userID).Count(&count)
	return count, result.Error
}

func (r *GormRepo) CreateItem(item *CharacterItem) error {
	return r.db.Create(item).Error
}

func (r *GormRepo) DeleteItem(itemID uint) error {
	return r.db.Delete(&CharacterItem{}, itemID).Error
}

func (r *GormRepo) GetItemsByCharacterID(charID uint) ([]*CharacterItem, error) {
	var items []*CharacterItem
	result := r.db.Where("character_id = ?", charID).Find(&items)
	return items, result.Error
}

func (r *GormRepo) SetEquipment(charID uint, slotID uint8, itemID uint) error {
	eq := CharacterEquipment{CharacterID: charID, SlotID: slotID, ItemID: itemID}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "character_id"}, {Name: "slot_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"item_id"}),
	}).Create(&eq).Error
}

func (r *GormRepo) ClearEquipment(charID uint, slotID uint8) error {
	return r.db.Where("character_id = ? AND slot_id = ?", charID, slotID).
		Delete(&CharacterEquipment{}).Error
}

func (r *GormRepo) GetEquipment(charID uint) ([]*CharacterEquipment, error) {
	var eqs []*CharacterEquipment
	result := r.db.Where("character_id = ?", charID).Find(&eqs)
	return eqs, result.Error
}

func (r *GormRepo) UpsertLoadout(charID uint, slots [6]string) error {
	lo := CharacterLoadout{CharacterID: charID}
	result := r.db.Where("character_id = ?", charID).Assign(CharacterLoadout{
		Slot0: slots[0],
		Slot1: slots[1],
		Slot2: slots[2],
		Slot3: slots[3],
		Slot4: slots[4],
		Slot5: slots[5],
	}).FirstOrCreate(&lo)
	return result.Error
}

func (r *GormRepo) GetLoadout(charID uint) (*CharacterLoadout, error) {
	var lo CharacterLoadout
	result := r.db.Where("character_id = ?", charID).First(&lo)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &lo, result.Error
}

func (r *GormRepo) UpsertFluxCommitment(charID uint, entries []FluxCommitmentEntry) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing rows for this character.
		if err := tx.Where("character_id = ?", charID).Delete(&CharacterFluxCommitment{}).Error; err != nil {
			return err
		}
		// Insert new rows.
		for _, e := range entries {
			row := CharacterFluxCommitment{
				CharacterID: charID,
				School:      e.School,
				Percentage:  e.Percentage,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GormRepo) GetFluxCommitment(charID uint) ([]FluxCommitmentEntry, error) {
	var rows []CharacterFluxCommitment
	result := r.db.Where("character_id = ?", charID).Find(&rows)
	if result.Error != nil {
		return nil, result.Error
	}
	entries := make([]FluxCommitmentEntry, len(rows))
	for i, row := range rows {
		entries[i] = FluxCommitmentEntry{School: row.School, Percentage: row.Percentage}
	}
	return entries, nil
}
