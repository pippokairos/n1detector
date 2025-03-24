package example

type User struct {
	ID        uint
	Languages []Language `gorm:"many2many:user_languages;"`
}

type Language struct {
	ID   uint
	Name string
}
