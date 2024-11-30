package userRepo

import (
	"context"

	"github.com/ghaniswara/dating-app/internal/entity"
	"gorm.io/gorm"
)

type IUserRepo interface {
	CreateUser(ctx context.Context, user entity.User) (*entity.User, error)
	GetUserByID(ctx context.Context, id int) (*entity.User, error)
	GetUserByUnameOrEmail(ctx context.Context, email, uname string) (*entity.User, error)
}

type UserRepo struct {
	db *gorm.DB
}

func New(db *gorm.DB) IUserRepo {
	return &UserRepo{
		db: db,
	}
}

func (r *UserRepo) CreateUser(ctx context.Context, user entity.User) (*entity.User, error) {
	result := r.db.WithContext(ctx).Create(&user)
	return &user, result.Error
}

func (r *UserRepo) GetUserByID(ctx context.Context, id int) (*entity.User, error) {
	var user entity.User
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&user)
	return &user, result.Error
}

func (r *UserRepo) GetUserByUnameOrEmail(ctx context.Context, email, uname string) (*entity.User, error) {
	var user entity.User
	query := r.db.WithContext(ctx)
	if email != "" {
		query = query.Where("email = ?", email)
	}
	if uname != "" {
		query = query.Or("username = ?", uname)
	}
	result := query.First(&user)
	return &user, result.Error
}
