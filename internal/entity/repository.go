package entity

import "time"

// TODO Refactor premium to use payment transaction table
type User struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	Name      string    `gorm:"not null;column:name"`
	Email     string    `gorm:"unique;not null;column:email"`
	Username  string    `gorm:"unique;column:username"`
	Password  string    `gorm:"not null;column:password"`
	IsPremium bool      `gorm:"not null;column:is_premium"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamp;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamp;not null"`
}

type SwipeTransaction struct {
	ID     uint      `gorm:"primaryKey;column:id"`
	UserID uint      `gorm:"column:user_id;not null"`
	ToID   uint      `gorm:"column:to_id;not null"`
	Date   time.Time `gorm:"column:date;type:date;not null"`
	Action Action    `gorm:"column:action;type:smallint;not null"`
	Time   time.Time `gorm:"column:timestamp;type:timestamp;not null"`

	// Snapshot field, allow quick fetch of list of liked profiles
	// For fetching list of matched profiles
	IsMatched bool `gorm:"column:is_matched;not null"`
}

type Action uint

const (
	ActionLike Action = iota + 1
	ActionPass
	ActionSuperLike
)

func (a Action) String() string {
	switch a {
	case ActionLike:
		return "like"
	case ActionPass:
		return "pass"
	case ActionSuperLike:
		return "superlike"
	default:
		return "unknown"
	}
}

type LikedProfiles struct {
	ProfileIDs []int `json:"profile_ids"`
}

type Outcome uint

const (
	OutcomeMatch        Outcome = iota + 1 //When both user like each other
	OutcomeMissed                          //When one user pass the other user which likes the user
	OutcomeLimitReached                    //When user reach the maximum likes per day
	OutcomeNoLike                          //When user pass the other user without like
	OutcomeNotFound                        //When user not found
)

func (o Outcome) String() string {
	switch o {
	case OutcomeMatch:
		return "Match"
	case OutcomeMissed:
		return "Missed"
	case OutcomeLimitReached:
		return "Limit Reached"
	default:
		return "Unknown"
	}
}
