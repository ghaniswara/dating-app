package entity

type SignUpResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

type MatchSwipeResponse struct {
	Outcome     string  `json:"outcome"`
	OutcomeEnum Outcome `json:"outcome_enum"`
}

type MatchGetProfileResponse struct {
	Profiles []User `json:"profiles"`
}

type SignInResponse struct {
	Token string `json:"token"`
}
