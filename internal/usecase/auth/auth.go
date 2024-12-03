package authUseCase

import (
	"context"
	"net/http"
	"strings"

	"github.com/ghaniswara/dating-app/internal/entity"
	userRepo "github.com/ghaniswara/dating-app/internal/repository/user"
	"github.com/ghaniswara/dating-app/pkg/jwt"
	"github.com/labstack/echo"
	"golang.org/x/crypto/bcrypt"
)

type IAuthUseCase interface {
	SignupUser(ctx context.Context, request entity.CreateUserRequest) (*entity.User, error)
	SignIn(ctx context.Context, email, username, password string) (string, error)
	GetUserFromJWTRequest(c echo.Context) (*entity.User, error)
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

	user := entity.User{
		Name:      authData.Name,
		Email:     authData.Email,
		Username:  authData.Username,
		Password:  authData.Password,
		IsPremium: false,
	}

	return p.userRepo.CreateUser(ctx, &user)
}

func (p *authUseCase) SignIn(ctx context.Context, email, username, password string) (string, error) {
	user, err := p.userRepo.GetUserByUnameOrEmail(ctx, email, username)
	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password+user.Email)); err != nil {
		return "", err
	}

	token, err := jwt.CreateToken(int(user.ID), user.Username)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (p *authUseCase) GetUserFromJWTRequest(c echo.Context) (*entity.User, error) {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil, c.JSON(http.StatusUnauthorized, map[string]string{"message": "missing token"})
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, c.JSON(http.StatusUnauthorized, map[string]string{"message": "invalid token format"})
	}
	token := parts[1]

	claims, err := jwt.ValidateToken(token)

	if err != nil {
		return nil, c.JSON(http.StatusUnauthorized, map[string]string{"message": "invalid token"})
	}

	return p.userRepo.GetUserByID(c.Request().Context(), claims.UserID)
}
