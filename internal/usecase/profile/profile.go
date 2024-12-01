package profile

import (
	userRepo "github.com/ghaniswara/dating-app/internal/repository/user"
)

type iProfileUseCase interface {
}

type profileUseCase struct {
	userRepo userRepo.UserRepo
}

func NewProfileUseCase(userRepo *userRepo.UserRepo) iProfileUseCase {
	return &profileUseCase{
		userRepo: *userRepo,
	}
}
