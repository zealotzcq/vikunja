package models

import (
	"time"

	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"
	"xorm.io/xorm"
)

type ProjectTaskRefresh struct {
	ProjectID int64     `json:"project_id"`
	SinceTime time.Time `json:"since_time"`
}

func RefreshProjectTasks(s *xorm.Session, refresh *ProjectTaskRefresh, a web.Auth) (err error) {

	project := &Project{
		ID: refresh.ProjectID,
	}
	canRead, _, err := project.CanRead(s, a)
	if err != nil {
		return err
	}
	if !canRead {
		u, err := user.GetFromAuth(a)
		if err != nil {
			return err
		}
		return ErrNeedToHaveProjectReadAccess{ProjectID: refresh.ProjectID, UserID: u.ID}
	}

	var tasks []*Task
	err = s.Where("project_id = ? AND updated < ?", refresh.ProjectID, refresh.SinceTime).Find(&tasks)
	if err != nil {
		return err
	}

	refreshedCount := 0
	for _, task := range tasks {
		if task.Done {
			task.Done = false
			task.DoneAt = time.Time{}
			_, updateErr := s.ID(task.ID).Cols("done", "done_at").Update(task)
			if updateErr != nil {
				log.Errorf("Failed to mark task %d as incomplete: %s", task.ID, updateErr.Error())
				continue
			}
			refreshedCount++
		}
	}

	if refreshedCount > 0 {
		_, err = s.ID(project.ID).Cols("updated").Update(project)
		if err != nil {
			log.Errorf("Failed to update project last updated timestamp: %s", err.Error())
		}
	}

	log.Debugf("Refreshed %d tasks for project %d since %s", refreshedCount, refresh.ProjectID, refresh.SinceTime)

	return nil
}
