package routesV1Match

import (
	"net/http"
	"strconv"

	"github.com/ghaniswara/dating-app/internal/entity"
	authUseCase "github.com/ghaniswara/dating-app/internal/usecase/auth"
	"github.com/ghaniswara/dating-app/internal/usecase/match"
	"github.com/ghaniswara/dating-app/pkg/http_util"
	"github.com/labstack/echo"
)

func GetProfileHandler(c echo.Context, matchCase match.IMatchUseCase, authCase authUseCase.IAuthUseCase) error {
	request, err := http_util.Decode[entity.MatchGetProfileRequest](c)

	if err != nil {
		return http_util.Encode(c, http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	user, err := authCase.GetUserFromJWTRequest(c)

	if err != nil {
		return http_util.Encode(c, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
	}

	profiles, err := matchCase.GetDatingProfiles(c.Request().Context(), int(user.ID), request.ExcludeProfiles, 10)

	if err != nil {
		return http_util.Encode(c, http.StatusInternalServerError, map[string]string{"error": "failed to get profiles"})
	}

	return http_util.Encode(c, http.StatusOK, http_util.HTTPResponse[entity.MatchGetProfileResponse]{
		Message: "Profiles fetched successfully",
		Data: entity.MatchGetProfileResponse{
			Profiles: profiles,
		},
	})
}

func LikeHandler(c echo.Context, matchCase match.IMatchUseCase, authCase authUseCase.IAuthUseCase) error {
	likeRequest, err := http_util.Decode[entity.MatchLikeRequest](c)

	if err != nil {
		return http_util.Encode(c, http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	user, err := authCase.GetUserFromJWTRequest(c)

	if err != nil {
		return http_util.Encode(c, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
	}

	action := entity.ActionLike

	if likeRequest.IsSuperLike {
		action = entity.ActionSuperLike
	}

	likesToUserID, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		return http_util.Encode(c, http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	outcome, err := matchCase.SwipeDatingProfile(c.Request().Context(), int(user.ID), likesToUserID, action)

	if err != nil {
		return http_util.Encode(c, http.StatusInternalServerError, map[string]string{"error": "failed to swipe"})
	}

	return http_util.Encode(c, http.StatusOK, http_util.HTTPResponse[entity.MatchSwipeResponse]{
		Message: "Swipe outcome",
		Data: entity.MatchSwipeResponse{
			Outcome:     outcome.String(),
			OutcomeEnum: outcome,
		},
	})
}

func PassHandler(c echo.Context, matchCase match.IMatchUseCase, authCase authUseCase.IAuthUseCase) error {
	user, err := authCase.GetUserFromJWTRequest(c)

	if err != nil {
		return http_util.Encode(c, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
	}

	passToUserID, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		return http_util.Encode(c, http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	outcome, err := matchCase.SwipeDatingProfile(c.Request().Context(), int(user.ID), passToUserID, entity.ActionPass)

	if err != nil {
		return http_util.Encode(c, http.StatusInternalServerError, map[string]string{"error": "failed to swipe"})
	}

	return http_util.Encode(c, http.StatusOK, http_util.HTTPResponse[entity.MatchSwipeResponse]{
		Message: "Swipe outcome",
		Data: entity.MatchSwipeResponse{
			Outcome:     outcome.String(),
			OutcomeEnum: outcome,
		},
	})
}
