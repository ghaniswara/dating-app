package authUseCase

import (
	"context"

	"github.com/ghaniswara/dating-app/internal/entity"
	userRepo "github.com/ghaniswara/dating-app/internal/repository/user"
	"github.com/ghaniswara/dating-app/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

type IAuthUseCase interface {
	SignupUser(ctx context.Context, request entity.CreateUserRequest) (*entity.User, error)
	SignIn(ctx context.Context, email, username, password string) (string, error)
}

type authUseCase struct {
	userRepo userRepo.IUserRepo
}

func New(userRepo userRepo.IUserRepo) IAuthUseCase {
	return &authUseCase{
		userRepo: userRepo,
	}
}

func (p *authUseCase) SignupUser(ctx context.Context, authData entity.CreateUserRequest) (*entity.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(authData.Password+authData.Email), 12)
	if err != nil {
		return nil, err
	}
	authData.Password = string(hashedPassword)

	var user entity.User

	return p.userRepo.CreateUser(ctx, user)
}

func (p *authUseCase) SignIn(ctx context.Context, email, username, password string) (string, error) {
	user, err := p.userRepo.GetUserByUnameOrEmail(ctx, username, password)
	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password+user.Email)); err != nil {
		return "", err
	}

	token, err := jwt.CreateToken(user.UserName, user.Email)
	if err != nil {
		return "", err
	}
	return token, nil
}
