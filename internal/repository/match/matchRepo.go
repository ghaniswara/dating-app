package matchRepo

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/ghaniswara/dating-app/internal/entity"
	"github.com/go-redis/redis"

	"gorm.io/gorm"
)

type IMatchRepo interface {
	// User Table
	GetDatingProfiles(ctx context.Context, userID int, excludeIDs []int, limit int) ([]entity.User, error)

	// SwipeTransaction Table

	GetTodayLikesCount(ctx context.Context, userID int) (int, error)
	GetTodayLikedProfilesIDs(ctx context.Context, userID int) ([]int, error)

	// Query SwipeTransaction Table returning IDs that matched (like each other)
	GetMatchedProfilesIDs(ctx context.Context, userID int) ([]int, error)

	// Query SwipeTransaction Table returning IDs that swiped by the user with any action
	GetSwipedProfilesIDs(ctx context.Context, userID int, date *time.Time) ([]entity.SwipeTransaction, error)

	CreateSwipe(ctx context.Context, userID int, likedToUserID int, action entity.Action) (Outcome entity.Outcome, err error)
}

type MatchRepo struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewMatchRepo(db *gorm.DB, redis *redis.Client) IMatchRepo {
	return &MatchRepo{
		db:  db,
		rdb: redis,
	}
}

func (m *MatchRepo) GetTodayLikesCount(ctx context.Context, userID int) (int, error) {
	countKey := ":user:" + strconv.Itoa(userID) + ":likes:count"
	exists, err := m.rdb.Exists(countKey).Result()

	var count int

	if err != nil {
		return 0, err
	}

	if exists == 0 {
		count, err := m.getLikesCount(ctx, userID, time.Now())
		if err != nil {
			return 0, err
		}
		m.rdb.Set(countKey, count, getTTL())
	} else {
		count, err = m.rdb.Get(countKey).Int()
		if err != nil {
			return 0, err
		}
	}

	return count, nil
}

func (m *MatchRepo) GetTodayLikedProfilesIDs(ctx context.Context, userID int) ([]int, error) {
	profilesKey := ":user:" + strconv.Itoa(userID) + ":likes:profiles"

	var profiles []int

	exists, err := m.rdb.Exists(profilesKey).Result()

	if err != nil {
		return nil, err
	}

	now := time.Now()

	if exists == 0 {
		profiles, err = m.getLikedProfilesIDs(ctx, userID, &now)
		if err != nil {
			return nil, err
		}

		for _, v := range profiles {
			m.rdb.SAdd(profilesKey, v)
		}

		m.rdb.Expire(profilesKey, getTTL())
	} else {
		err = m.rdb.SMembers(profilesKey).ScanSlice(&profiles)
		if err != nil {
			return nil, err
		}
	}

	return profiles, nil
}

// TODO
// Refactor to use join table with the SwipeTransaction table
// With the new table, we can get the ranking of the user
func (m *MatchRepo) GetDatingProfiles(ctx context.Context, userID int, excludeProfiles []int, limit int) ([]entity.User, error) {
	var profiles []entity.User

	// Create a subquery to select random IDs
	subquery := m.db.WithContext(ctx).
		Model(&entity.User{}).
		Select("id").
		Where("id NOT IN ?", append(excludeProfiles, userID)).
		Order("RANDOM()").
		Limit(limit + 10)

	res := m.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id IN (?)", subquery).
		Find(&profiles)

	return profiles, res.Error
}

func (m *MatchRepo) CreateSwipe(ctx context.Context, userID int, likedToUserID int, action entity.Action) (entity.Outcome, error) {
	var pair *entity.SwipeTransaction
	// Check if liked profile exists
	var user *entity.User
	likedProfileRes := m.db.
		WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", likedToUserID).
		First(&user)

	if likedProfileRes.Error != nil {
		if likedProfileRes.Error == gorm.ErrRecordNotFound {
			return entity.OutcomeNotFound, nil
		}

		return 0, likedProfileRes.Error
	}

	// Check if both profile like each other
	if action == entity.ActionLike || action == entity.ActionSuperLike {
		m.appendLikedCountCacheToday(ctx, userID, 1)
		m.appendLikedProfilesCacheToday(ctx, userID, []int{likedToUserID})

		resPair := m.db.WithContext(ctx).
			Model(&entity.SwipeTransaction{}).
			Where("user_id = ? AND to_id = ? AND action = ?", likedToUserID, userID, entity.ActionLike).
			First(&pair)

		if resPair.Error != nil && resPair.Error != gorm.ErrRecordNotFound {
			return 0, resPair.Error
		}
	}

	// Create like transaction for the user
	isMatched := action == entity.ActionLike && pair.ID != 0

	res := m.db.WithContext(ctx).
		Model(&entity.SwipeTransaction{}).
		Create(&entity.SwipeTransaction{
			UserID:    uint(userID),
			ToID:      uint(likedToUserID),
			Date:      time.Now(),
			Action:    action,
			Time:      time.Now(),
			IsMatched: isMatched,
		})

	if res.Error != nil {
		return 0, res.Error
	}
	// update the pair to isMatched if both profile like each other
	if pair != nil && action == entity.ActionLike {
		res := m.db.WithContext(ctx).Model(&entity.SwipeTransaction{}).Where("user_id = ? AND to_id = ?", likedToUserID, userID).Update("is_matched", true)
		if res.Error != nil {
			return 0, res.Error
		}
	}

	isPairFound := pair != nil && pair.ID != 0

	if isPairFound && (action == entity.ActionLike || action == entity.ActionSuperLike) {
		m.appendMatchProfilesCache(ctx, userID, []int{likedToUserID})
		return entity.OutcomeMatch, nil
	}

	if isPairFound && action == entity.ActionPass {
		return entity.OutcomeMissed, nil
	}

	return entity.OutcomeNoLike, nil
}

func (m *MatchRepo) GetMatchedProfilesIDs(ctx context.Context, userID int) ([]int, error) {
	profilesKey := ":user:" + strconv.Itoa(userID) + ":match:profiles"

	var profiles []int

	exists, err := m.rdb.Exists(profilesKey).Result()

	if err != nil {
		return nil, err
	}

	if exists == 0 {
		res := m.db.WithContext(ctx).
			Model(&entity.SwipeTransaction{}).
			Select("to_id").
			Where("user_id = ? AND is_matched = ?", userID, true).
			Find(&profiles)

		if err := m.rdb.SAdd(profilesKey, profiles).Err(); err != nil {
			log.Println("error adding match profiles to redis", err)
		}
		m.rdb.Expire(profilesKey, 30*24*time.Hour)

		return profiles, res.Error
	} else {
		err = m.rdb.SMembers(profilesKey).ScanSlice(&profiles)

		if err != nil {
			return nil, err
		}
	}

	return profiles, nil
}

func (m *MatchRepo) GetSwipedProfilesIDs(ctx context.Context, userID int, date *time.Time) ([]entity.SwipeTransaction, error) {
	var profiles []entity.SwipeTransaction
	query := m.db.WithContext(ctx).
		Model(&entity.SwipeTransaction{}).
		Select("user_id, to_id, action").
		Where("user_id = ?", userID)

	if date != nil {
		query = query.Where("date = ?", *date)
	}

	res := query.Find(&profiles)

	return profiles, res.Error
}

// Private functions

func (m *MatchRepo) getLikesCount(ctx context.Context, userID int, date time.Time) (int, error) {
	var count int64
	res := m.db.WithContext(ctx).
		Model(&entity.SwipeTransaction{}).
		Where("user_id = ? AND date = ? ", userID, date).
		Count(&count)

	return int(count), res.Error
}

func (m *MatchRepo) getLikedProfilesIDs(ctx context.Context, userID int, date *time.Time) ([]int, error) {
	var profiles []int
	query := m.db.WithContext(ctx).
		Model(&entity.SwipeTransaction{}).
		Select("to_id").
		Where("user_id = ?", userID)

	if date != nil {
		query = query.Where("date = ?", *date)
	}

	res := query.Find(&profiles)

	return profiles, res.Error
}

func (m *MatchRepo) appendLikedCountCacheToday(_ context.Context, userID int, count int) error {
	countKey := ":user:" + strconv.Itoa(userID) + ":likes:count"

	return m.rdb.IncrBy(countKey, int64(count)).Err()
}

func (m *MatchRepo) appendLikedProfilesCacheToday(_ context.Context, userID int, profiles []int) error {
	profilesKey := ":user:" + strconv.Itoa(userID) + ":likes:profiles"

	return m.rdb.SAdd(profilesKey, profiles).Err()
}

func (m *MatchRepo) appendMatchProfilesCache(_ context.Context, userID int, newProfiles []int) error {
	profilesKey := ":user:" + strconv.Itoa(userID) + ":match:profiles"
	var currentProfiles []int

	err := m.rdb.SMembers(profilesKey).ScanSlice(&currentProfiles)
	if err != nil && err != redis.Nil {
		return err
	}

	currentProfiles = append(currentProfiles, newProfiles...)

	if err := m.rdb.SAdd(profilesKey, currentProfiles).Err(); err != nil {
		log.Println("error updating match profiles in redis", err)
		return err
	}

	m.rdb.Expire(profilesKey, 30*24*time.Hour)

	return nil
}

// Helper

func getTTL() time.Duration {
	now := time.Now()
	startOfTomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	ttlBeforeTomorrow := startOfTomorrow.Add(24 * time.Hour).Sub(now)

	return ttlBeforeTomorrow
}
