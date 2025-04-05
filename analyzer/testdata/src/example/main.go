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
		err := repo.SetUserLocation(user.ID, locationID) // want "Potential N\\+1 query detected: call to example.SetUserLocation may lead to DB query inside loop"
		if err != nil {
			return err
		}
	}
	return nil
}
