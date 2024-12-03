package match

import (
	"context"

	"github.com/ghaniswara/dating-app/internal/entity"
	matchRepo "github.com/ghaniswara/dating-app/internal/repository/match"
	userRepo "github.com/ghaniswara/dating-app/internal/repository/user"
	"github.com/go-redis/redis"
)

type IMatchUseCase interface {
	GetDatingProfiles(ctx context.Context, userID int, excludeProfiles []int, limit int) ([]entity.User, error)
	SwipeDatingProfile(ctx context.Context, userID int, likedToUserID int, action entity.Action) (entity.Outcome, error)
}

type matchUseCase struct {
	userRepo  userRepo.IUserRepo
	matchRepo matchRepo.IMatchRepo
}

func NewMatchUseCase(userRepo userRepo.IUserRepo, redisCache *redis.Client, matchRepo matchRepo.IMatchRepo) IMatchUseCase {
	return &matchUseCase{
		userRepo:  userRepo,
		matchRepo: matchRepo,
	}
}

func (m *matchUseCase) GetDatingProfiles(ctx context.Context, userID int, excludeProfiles []int, limit int) ([]entity.User, error) {
	likedProfiles, err := m.matchRepo.GetTodayLikedProfiles(ctx, userID)

	if err != nil {
		return nil, err
	}

	matchedProfiles, err := m.matchRepo.GetMatchProfiles(ctx, userID)

	if err != nil {
		return nil, err
	}

	excludeProfiles = append(excludeProfiles, likedProfiles...)
	excludeProfiles = append(excludeProfiles, matchedProfiles...)

	profiles, err := m.matchRepo.GetDatingProfiles(ctx, userID, excludeProfiles, limit)

	if err != nil {
		return nil, err
	}

	return profiles, nil
}

// TODO Implement premium feature
// TODO Implement super like feature
// TODO Implement you missed feature
func (m *matchUseCase) SwipeDatingProfile(
	ctx context.Context,
	userID int,
	likedToUserID int,
	action entity.Action,
) (entity.Outcome, error) {

	likesCount, err := m.matchRepo.GetTodayLikesCount(ctx, userID)

	if err != nil {
		return 0, err
	}

	// TODO: implement premium feature
	user, err := m.userRepo.GetUserByID(ctx, likedToUserID)

	if err != nil {
		return 0, err
	}

	if likesCount >= 10 && !user.Premium {
		return entity.OutcomeLimitReached, nil
	}

	Outcome, err := m.matchRepo.CreateSwipe(ctx, userID, likedToUserID, action)

	if err != nil {
		return 0, err
	}

	if Outcome == entity.OutcomeMatch {
		return entity.OutcomeMatch, nil
	}

	return Outcome, nil
}
