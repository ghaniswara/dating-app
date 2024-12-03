package routesV1Match

import (
	"net/http"
	"strconv"

	"github.com/ghaniswara/dating-app/internal/entity"
	"github.com/ghaniswara/dating-app/internal/usecase/match"
	serializer "github.com/ghaniswara/dating-app/pkg/http_util"
	"github.com/labstack/echo"
)

type MatchLikeRequest struct {
	IsSuperLike bool `json:"is_super_like"`
}

type MatchGetProfileRequest struct {
	ExcludeProfiles []int `json:"exclude_profiles"`
}

type MatchGetProfileResponse struct {
	Profiles []entity.User `json:"profiles"`
}

type MatchSwipeResponse struct {
	Outcome string `json:"outcome"`
}

func GetProfileHandler(c echo.Context, matchCase match.IMatchUseCase) error {
	request, err := serializer.Decode[MatchGetProfileRequest](c)

	if err != nil {
		return serializer.Encode(c, http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	user := c.Request().Context().Value("userProfile").(*entity.User)

	profiles, err := matchCase.GetDatingProfiles(c.Request().Context(), int(user.ID), request.ExcludeProfiles, 10)

	if err != nil {
		return serializer.Encode(c, http.StatusInternalServerError, map[string]string{"error": "failed to get profiles"})
	}

	return serializer.Encode(c, http.StatusOK, MatchGetProfileResponse{Profiles: profiles})
}

func LikeHandler(c echo.Context, matchCase match.IMatchUseCase) error {
	likeRequest, err := serializer.Decode[MatchLikeRequest](c)

	if err != nil {
		return serializer.Encode(c, http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	user := c.Request().Context().Value("userProfile").(*entity.User)

	action := entity.ActionLike

	if likeRequest.IsSuperLike {
		action = entity.ActionSuperLike
	}

	likesToUserID, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		return serializer.Encode(c, http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	outcome, err := matchCase.SwipeDatingProfile(c.Request().Context(), int(user.ID), likesToUserID, action)

	if err != nil {
		return serializer.Encode(c, http.StatusInternalServerError, map[string]string{"error": "failed to swipe"})
	}

	return serializer.Encode(c, http.StatusOK, MatchSwipeResponse{Outcome: outcome.String()})
}

func PassHandler(c echo.Context, matchCase match.IMatchUseCase) error {
	user := c.Request().Context().Value("userProfile").(*entity.User)

	passToUserID, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		return serializer.Encode(c, http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	outcome, err := matchCase.SwipeDatingProfile(c.Request().Context(), int(user.ID), passToUserID, entity.ActionPass)

	if err != nil {
		return serializer.Encode(c, http.StatusInternalServerError, map[string]string{"error": "failed to swipe"})
	}

	return serializer.Encode(c, http.StatusOK, MatchSwipeResponse{Outcome: outcome.String()})
}
