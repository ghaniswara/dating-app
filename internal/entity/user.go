package entity

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `gorm:"not null"`
	Email    string `gorm:"unique;not null"`
	Username string `gorm:"unique"`
	Password string `gorm:"not null"`
}
