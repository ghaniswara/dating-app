package match__test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/ghaniswara/dating-app/internal/entity"
	matchRepository "github.com/ghaniswara/dating-app/internal/repository/match"
	"github.com/ghaniswara/dating-app/pkg/http_util"
	helper_test "github.com/ghaniswara/dating-app/test/helper"
	"github.com/go-faker/faker/v4"
	"gotest.tools/assert"
)

var globalResources *helper_test.TestServerResources

func TestMain(m *testing.M) {
	// Set up the test server
	resources, err := helper_test.SetupTestServer(context.TODO())
	var code int

	if err != nil {
		log.Printf("Failed to set up test server: %s", err)
		code = 1
	} else {
		// Run tests
		globalResources = resources
		code = m.Run()
	}

	resources.CleanupTestServer()
	os.Exit(code)
}

// Create 5 profile, like all of them
// check transaction table, there should be 5 record with action = 1
func TestLike(t *testing.T) {
	users, err := helper_test.PopulateUsers(globalResources.ORM, 5)

	if err != nil {
		t.Fatalf("Failed to populate users: %s", err)
	}

	username := faker.Username()
	password := faker.Password()
	email := faker.Email()

	user, err := helper_test.SignUpUser(t, username, password, email)
	if err != nil {
		t.Fatalf("Failed to sign up user: %s", err)
	}

	token, err := helper_test.SignInUser(t, email, username, password)

	if err != nil {
		t.Fatalf("Failed to create token: %s", err)
	}

	var OutcomeCount map[entity.Outcome]int = make(map[entity.Outcome]int)

	for _, v := range users {
		response := createMatchRequest(t, token, v.ID, entity.ActionLike)
		fmt.Printf("Response: %+v\n", response)

		OutcomeCount[response.OutcomeEnum]++
	}

	// Check with repository
	matchRepo := matchRepository.NewMatchRepo(
		globalResources.ORM,
		globalResources.Redis,
	)

	likedProfiles, err := matchRepo.GetTodayLikedProfilesIDs(context.TODO(), user.ID)

	if err != nil {
		t.Fatalf("Failed to get today liked profiles: %s", err)
	}

	likedCount, err := matchRepo.GetTodayLikesCount(context.TODO(), user.ID)

	if err != nil {
		t.Fatalf("Failed to get today liked count: %s", err)
	}

	swipeTransaction, err := matchRepo.GetSwipedProfilesIDs(context.TODO(), user.ID, nil)

	if err != nil {
		t.Fatalf("Failed to get today swiped profiles: %s", err)
	}

	assert.Equal(t, len(likedProfiles), 5)
	assert.Equal(t, OutcomeCount[entity.OutcomeNoLike], 5)
	assert.Equal(t, likedCount, 5)
	assert.Equal(t, len(swipeTransaction), 5)

	for _, v := range swipeTransaction {
		assert.Equal(t, v.Action, entity.ActionLike)
	}
}

func TestPass(t *testing.T) {
	profiles, err := helper_test.PopulateUsers(globalResources.ORM, 5)

	matchRepo := matchRepository.NewMatchRepo(
		globalResources.ORM,
		globalResources.Redis,
	)

	username := faker.Username()
	password := faker.Password()
	email := faker.Email()

	user, err := helper_test.SignUpUser(t, username, password, email)
	if err != nil {
		t.Fatalf("Failed to sign up user: %s", err)
	}

	token, err := helper_test.SignInUser(t, email, username, password)

	if err != nil {
		t.Fatalf("Failed to create token: %s", err)
	}

	var OutcomeCount map[entity.Outcome]int = make(map[entity.Outcome]int)

	for _, v := range profiles {
		response := createMatchRequest(t, token, v.ID, entity.ActionPass)

		OutcomeCount[response.OutcomeEnum]++
	}

	transactions, err := matchRepo.GetSwipedProfilesIDs(context.TODO(), user.ID, nil)
	if err != nil {
		t.Fatalf("Failed to get swipe transactions: %s", err)
	}

	if len(transactions) != 5 {
		t.Errorf("Expected 5 transactions, got %d", len(transactions))
	}

	for _, transaction := range transactions {
		if transaction.Action != entity.ActionPass {
			t.Errorf("Expected all transactions to have action Pass, got %s", transaction.Action.String())
		}
	}
}

func TestMatch(t *testing.T) {
	// Create a user1 using the test_helper
	username := faker.Username()
	password := faker.Password()
	email := faker.Email()

	user1, err := helper_test.SignUpUser(t, username, password, email)
	if err != nil {
		t.Fatalf("Failed to sign up user: %s", err)
	}

	token1, err := helper_test.SignInUser(t, email, username, password)
	if err != nil {
		t.Fatalf("Failed to sign in user: %s", err)
	}

	// Create a user2 using the test_helper
	username2 := faker.Username()
	password2 := faker.Password()
	email2 := faker.Email()

	user2, err := helper_test.SignUpUser(t, username2, password2, email2)
	if err != nil {
		t.Fatalf("Failed to sign up user: %s", err)
	}

	token2, err := helper_test.SignInUser(t, email2, username2, password2)
	if err != nil {
		t.Fatalf("Failed to sign in user: %s", err)
	}

	resp1 := createMatchRequest(t, token1, uint(user2.ID), entity.ActionLike)
	resp2 := createMatchRequest(t, token2, uint(user1.ID), entity.ActionLike)

	matchRepo := matchRepository.NewMatchRepo(
		globalResources.ORM,
		globalResources.Redis,
	)

	matchedProfiles1, err := matchRepo.GetMatchedProfilesIDs(context.TODO(), user1.ID)
	if err != nil {
		t.Fatalf("Failed to get matched profiles: %s", err)
	}

	matchedProfiles2, err := matchRepo.GetMatchedProfilesIDs(context.TODO(), user2.ID)
	if err != nil {
		t.Fatalf("Failed to get matched profiles: %s", err)
	}

	assert.Equal(t, len(matchedProfiles1), 1)
	assert.Equal(t, len(matchedProfiles2), 1)
	assert.Equal(t, matchedProfiles1[0], (user2.ID))
	assert.Equal(t, matchedProfiles2[0], (user1.ID))
	assert.Equal(t, resp1.OutcomeEnum, entity.OutcomeNoLike)
	assert.Equal(t, resp2.OutcomeEnum, entity.OutcomeMatch)
}

func TestLikeLimit(t *testing.T) {
	// Create 11 profiles
	profiles, err := helper_test.PopulateUsers(globalResources.ORM, 11)
	if err != nil {
		t.Fatalf("Failed to populate profiles: %s", err)
	}

	// Create a user using the test_helper
	username := faker.Username()
	password := faker.Password()
	email := faker.Email()

	user, err := helper_test.SignUpUser(
		t,
		username,
		password,
		email,
	)
	if err != nil {
		t.Fatalf("Failed to sign up user: %s", err)
	}

	token, err := helper_test.SignInUser(t, email, username, password)
	if err != nil {
		t.Fatalf("Failed to sign in user: %s", err)
	}

	// Create the matchRepo
	matchRepo := matchRepository.NewMatchRepo(
		globalResources.ORM,
		globalResources.Redis,
	)

	// Like all of them except the last one
	for _, v := range profiles[:len(profiles)-1] {
		createMatchRequest(t, token, v.ID, entity.ActionLike)
	}

	// Request should be failed at the last one
	response := createMatchRequest(t, token, profiles[len(profiles)-1].ID, entity.ActionLike)
	if response.OutcomeEnum != entity.OutcomeLimitReached {
		t.Errorf("Expected the last like to fail due to limit, but it succeeded")
	}

	likesCount, err := matchRepo.GetTodayLikesCount(context.TODO(), user.ID)
	if err != nil {
		t.Fatalf("Failed to get today likes count: %s", err)
	}

	assert.Equal(t, likesCount, 10)
}

// TODO : refactor for parallel test (go test ./test/match/*)
// Will fail on parallel test
func TestNoSameProfile(t *testing.T) {
	// Create 10 profiles
	profiles, err := helper_test.PopulateUsers(globalResources.ORM, 10)
	if err != nil {
		t.Fatalf("Failed to populate profiles: %s", err)
	}

	// Create a user using the test_helper
	username := faker.Username()
	password := faker.Password()
	email := faker.Email()

	_, err = helper_test.SignUpUser(t, username, password, email)

	if err != nil {
		t.Fatalf("Failed to sign up user: %s", err)
	}

	token, err := helper_test.SignInUser(t, email, username, password)

	if err != nil {
		t.Fatalf("Failed to sign in user: %s", err)
	}

	// Like 2 User
	for _, v := range profiles[:2] {
		createMatchRequest(t, token, v.ID, entity.ActionLike)
	}

	// Pass 2 User
	for _, v := range profiles[2:4] {
		createMatchRequest(t, token, v.ID, entity.ActionPass)
	}

	matchProfiles, err := getMatchProfiles(t, token, nil)

	if err != nil {
		t.Fatalf("Failed to get profiles: %s", err)
	}

	assert.Equal(t, len(matchProfiles), 6)

}

func createMatchRequest(t *testing.T, token string, profileID uint, method entity.Action) entity.MatchSwipeResponse {
	action := method.String()

	if method == entity.ActionSuperLike {
		action = entity.ActionLike.String()
	}

	requestURL := fmt.Sprintf("http://localhost:8080/v1/match/profile/%d/%s", profileID, action)

	req, err := http.NewRequest(http.MethodPost, requestURL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %s", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if method == entity.ActionSuperLike {
		reqBody, err := json.Marshal(entity.MatchLikeRequest{
			IsSuperLike: true,
		})
		if err != nil {
			t.Fatalf("Failed to marshal request body: %s", err)
		}
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		t.Fatalf("Failed to send request: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Logf("Response: %v", resp)
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	response := http_util.HTTPErrorResponse[entity.MatchSwipeResponse]{}
	response, err = http_util.DecodeBody[http_util.HTTPErrorResponse[entity.MatchSwipeResponse]](bodyBytes, response)
	if err != nil {
		t.Fatalf("Failed to decode response: %s", err)
	}

	return response.Data
}

func getMatchProfiles(t *testing.T, token string, excludeIDs []int) ([]entity.User, error) {
	requestURL := "http://localhost:8080/v1/match/profile"

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %s", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	reqBody, err := json.Marshal(entity.MatchGetProfileRequest{
		ExcludeProfiles: excludeIDs,
	})
	if err != nil {
		t.Fatalf("Failed to marshal request body: %s", err)
	}
	req.Body = io.NopCloser(bytes.NewBuffer(reqBody))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	response := http_util.HTTPResponse[entity.MatchGetProfileResponse]{}
	response, err = http_util.DecodeBody(bodyBytes, response)

	if err != nil {
		t.Fatalf("Failed to decode response: %s", err)
	}

	return response.Data.Profiles, nil
}
