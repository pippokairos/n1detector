package example

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type DB struct {
	*gorm.DB
}

type Repository struct {
	db       *DB
	timezone string
}

func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) PatchUsers(users []User) error {
	_, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()

	for idx := range users {
		if errs := errors.Join(
			r.db.Model(&users[idx]).Omit("Languages.*").Association("Languages").Replace(users[idx].Languages),           // want `Potential N\+1 query detected: DB query inside a loop`
			r.db.Model(&users[idx]).Select("LocationID", "EndDate", "SSO", "UpdatedByUserID").Updates(&users[idx]).Error, // want `Potential N\+1 query detected: DB query inside a loop` `Potential N\+1 query detected: DB query inside a loop`

		); errs != nil {
			return errs
		}
	}
	return nil
}

func (r *Repository) SetUserLocation(userID uint, locationID uint) error {
	return r.db.Model(&User{}).Where("id = ?", userID).Update("location_id", locationID).Error
}
