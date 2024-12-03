package entity

import (
	"context"
	"regexp"
)

type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}

func (r *CreateUserRequest) Validate(ctx context.Context) (problems map[string][]string) {
	problems = make(map[string][]string)

	if r.Name == "" {
		problems["Name"] = append(problems["Name"], "Name is required")
	}
	if r.Email == "" {
		problems["Email"] = append(problems["Email"], "Email is required")
	}

	if r.Username == "" {
		problems["Username"] = append(problems["Username"], "Username is required")
	}

	if len(r.Username) > 16 {
		problems["Username"] = append(problems["Username"], "User name is too long")
	}

	if r.Password == "" {
		problems["Password"] = append(problems["Password"], "Password is required")
	}

	if len([]byte(r.Password)) > 72 {
		problems["Password"] = append(problems["Password"], "Password length should not exceed 72 bytes")
	}

	return problems
}

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}

func (r *SignInRequest) Validate(ctx context.Context) (problems map[string][]string) {
	problems = make(map[string][]string)

	if r.Email == "" && r.Username == "" {
		problems["Email/Username"] = append(problems["Email/Username"], "Either Email or Username is required")
	}

	if r.Email != "" {
		emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
		if !regexp.MustCompile(emailRegex).MatchString(r.Email) {
			problems["Email"] = append(problems["Email"], "Invalid email format")
		}
	}

	if r.Password == "" {
		problems["Password"] = append(problems["Password"], "Password is required")
	}

	return problems
}

type MatchLikeRequest struct {
	IsSuperLike bool `json:"is_super_like"`
}

type MatchGetProfileRequest struct {
	ExcludeProfiles []int `json:"exclude_profiles"`
}
