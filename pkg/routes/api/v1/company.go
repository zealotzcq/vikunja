package v1

import (
	"net/http"

	"code.vikunja.io/api/pkg/company"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/auth"

	"github.com/labstack/echo/v5"
)

// GetUserCompanies retrieves all companies the user belongs to
func GetUserCompanies(c *echo.Context) error {
	a, err := auth.GetAuthFromClaims(c)
	if err != nil {
		return err
	}

	if _, is := a.(*models.LinkSharing); is {
		return echo.ErrForbidden
	}

	userID := a.GetID()

	s := db.NewSession()
	defer s.Close()

	companies, err := company.GetUserCompanies(s, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, companies)
}

// GetUserRelationsAsSubordinate retrieves all company relations where the user is a subordinate
func GetUserRelationsAsSubordinate(c *echo.Context) error {
	a, err := auth.GetAuthFromClaims(c)
	if err != nil {
		return err
	}

	if _, is := a.(*models.LinkSharing); is {
		return echo.ErrForbidden
	}

	userID := a.GetID()

	s := db.NewSession()
	defer s.Close()

	relations, err := company.GetUserRelationsAsSubordinate(s, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, relations)
}

// GetUserCompanyProjectMap retrieves a map of company IDs to project IDs for the current user
func GetUserCompanyProjectMap(c *echo.Context) error {
	a, err := auth.GetAuthFromClaims(c)
	if err != nil {
		return err
	}

	if _, is := a.(*models.LinkSharing); is {
		return echo.ErrForbidden
	}

	userID := a.GetID()

	s := db.NewSession()
	defer s.Close()

	projectMap, err := company.GetUserCompanyProjectMap(s, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, projectMap)
}
