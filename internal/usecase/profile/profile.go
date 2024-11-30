package profile

import (
	"context"

	"github.com/ghaniswara/dating-app/internal/entity"
	userRepo "github.com/ghaniswara/dating-app/internal/repository/user"
	"golang.org/x/crypto/bcrypt"
)

type iProfileUseCase interface {
	SignupUser(ctx context.Context, user entity.User) (*entity.User, error)
}

type profileUseCase struct {
	userRepo userRepo.UserRepo
}

func NewProfileUseCase(userRepo *userRepo.UserRepo) iProfileUseCase {
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
