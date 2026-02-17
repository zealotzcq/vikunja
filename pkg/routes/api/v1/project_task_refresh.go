package v1

import (
	"net/http"
	"strconv"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/auth"
	"github.com/labstack/echo/v5"
)

func RefreshProjectTasks(c *echo.Context) error {
	projectID := c.Param("project")

	refresh := &models.ProjectTaskRefresh{}
	if err := c.Bind(&refresh); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request payload")
	}

	id, err := strconv.ParseInt(projectID, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid project ID")
	}
	refresh.ProjectID = id

	authProvider, err := auth.GetAuthFromClaims(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Error getting auth info").Wrap(err)
	}

	s := db.NewSession()
	defer s.Close()

	if err := models.RefreshProjectTasks(s, refresh, authProvider); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, models.Message{
		Message: "Tasks refreshed successfully",
	})
}
