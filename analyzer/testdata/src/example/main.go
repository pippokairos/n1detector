package example

import (
	"gorm.io/gorm"
)

func ProcessUsers(users []User) error {
	db := &DB{&gorm.DB{}}
	repo := NewRepository(db)

	return repo.PatchUsers(users)
}

func SetLocation(users []User, locationID uint) error {
	db := &DB{&gorm.DB{}}
	repo := NewRepository(db)

	for _, user := range users {
		err := repo.SetUserLocation(user.ID, locationID) // TODO: want `Potential N\+1 query detected`
		if err != nil {
			return err
		}
	}
	return nil
}
