package profile

import (
	"context"

	"github.com/ghaniswara/dating-app/internal/entity"
	"github.com/ghaniswara/dating-app/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type iProfileUseCase interface {
	SignupUser(ctx context.Context, user entity.User) (*entity.User, error)
}

type profileUseCase struct {
	userRepo repository.UserRepo
}

func NewProfileUseCase(userRepo *repository.UserRepo) iProfileUseCase {
	return &profileUseCase{
		userRepo: *userRepo,
	}
}

func (p *profileUseCase) SignupUser(ctx context.Context, user entity.User) (*entity.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
	if err != nil {
		return nil, err
	}
	user.Password = string(hashedPassword)
	return p.userRepo.CreateUser(ctx, user)
}
