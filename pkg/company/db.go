// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package company

import (
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/log"
	"xorm.io/xorm"
)

var x *xorm.Engine

// SetEngine sets the xorm.Engine
func SetEngine() (err error) {
	x, err = db.CreateDBEngine()
	if err != nil {
		log.Criticalf("Could not connect to db: %v", err.Error())
		return
	}

	return nil
}

type Company struct {
	ID          int64  `xorm:"bigint autoincr not null unique pk" json:"id"`
	Description string `xorm:"varchar(500) not null" json:"description"`
	Created     int64  `xorm:"created not null" json:"created"`
	InviteCode  string `xorm:"varchar(100) not null unique" json:"invite_code"`
}

type CompanyStaff struct {
	ID        int64  `xorm:"bigint autoincr not null unique pk" json:"id"`
	CompanyID int64  `xorm:"bigint not null index" json:"company_id"`
	UserID    int64  `xorm:"bigint not null index" json:"user_id"`
	Role      string `xorm:"varchar(20) not null" json:"role"`
	Created   int64  `xorm:"created not null" json:"created"`
}

type CompanyRelation struct {
	ID                int64 `xorm:"bigint autoincr not null unique pk" json:"id"`
	CompanyID         int64 `xorm:"bigint not null index" json:"company_id"`
	SuperiorUserID    int64 `xorm:"bigint not null index" json:"superior_user_id"`
	SubordinateUserID int64 `xorm:"bigint not null index" json:"subordinate_user_id"`
	ProjectID         int64 `xorm:"bigint not null index" json:"project_id"`
	Created           int64 `xorm:"created not null" json:"created"`
	Updated           int64 `xorm:"updated not null" json:"updated"`
}

func GetTables() []interface{} {
	return []interface{}{
		&Company{},
		&CompanyStaff{},
		&CompanyRelation{},
	}
}

func GetCompanyByInviteCode(s *xorm.Session, inviteCode string) (*Company, error) {
	c := &Company{}
	has, err := s.Where("invite_code = ?", inviteCode).Get(c)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrInvalidInviteCode{inviteCode}
	}
	return c, nil
}

func AddStaffToCompany(s *xorm.Session, companyID, userID int64, role string) error {
	staff := &CompanyStaff{
		CompanyID: companyID,
		UserID:    userID,
		Role:      role,
	}
	_, err := s.Insert(staff)
	return err
}

func GetUserRole(s *xorm.Session, userID, companyID int64) string {
	staff := &CompanyStaff{}
	has, err := s.Where("user_id = ? AND company_id = ?", userID, companyID).Get(staff)
	if err != nil || !has {
		return ""
	}
	return staff.Role
}

type CompanyInfo struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	InviteCode  string `json:"invite_code"`
	Role        string `json:"role"`
}

func GetUserCompanies(s *xorm.Session, userID int64) ([]*CompanyInfo, error) {
	var staffList []*CompanyStaff
	err := s.Where("user_id = ?", userID).Find(&staffList)
	if err != nil {
		return nil, err
	}

	if len(staffList) == 0 {
		return []*CompanyInfo{}, nil
	}

	var companyIDs []int64
	for _, staff := range staffList {
		companyIDs = append(companyIDs, staff.CompanyID)
	}

	var companies []*Company
	err = s.In("id", companyIDs).Find(&companies)
	if err != nil {
		return nil, err
	}

	companyMap := make(map[int64]*Company)
	for _, comp := range companies {
		companyMap[comp.ID] = comp
	}

	var result []*CompanyInfo
	for _, staff := range staffList {
		if comp, exists := companyMap[staff.CompanyID]; exists {
			result = append(result, &CompanyInfo{
				ID:          comp.ID,
				Description: comp.Description,
				InviteCode:  comp.InviteCode,
				Role:        staff.Role,
			})
		}
	}

	return result, nil
}

type ErrInvalidInviteCode struct {
	InviteCode string
}

func (e ErrInvalidInviteCode) Error() string {
	return "Invalid invite code"
}

func (e ErrInvalidInviteCode) HTTPError() string {
	return "Invalid invite code"
}

func (e ErrInvalidInviteCode) HTTPCode() int {
	return 400
}

type CompanyRelationWithSuperior struct {
	ID                int64  `json:"id"`
	CompanyID         int64  `json:"company_id"`
	SuperiorUserID    int64  `json:"superior_user_id"`
	SubordinateUserID int64  `json:"subordinate_user_id"`
	ProjectID         int64  `json:"project_id"`
	Created           int64  `json:"created"`
	Updated           int64  `json:"updated"`
	SuperiorUsername  string `json:"superior_username"`
}

func GetUserRelationsAsSubordinate(s *xorm.Session, userID int64) ([]*CompanyRelationWithSuperior, error) {
	var relations []*CompanyRelation
	err := s.Where("subordinate_user_id = ?", userID).Find(&relations)
	if err != nil {
		return nil, err
	}

	if len(relations) == 0 {
		return []*CompanyRelationWithSuperior{}, nil
	}

	var superiorUserIDs []int64
	for _, rel := range relations {
		superiorUserIDs = append(superiorUserIDs, rel.SuperiorUserID)
	}

	var users []struct {
		ID       int64
		Username string
	}
	err = s.Table("users").In("id", superiorUserIDs).Find(&users)
	if err != nil {
		return nil, err
	}

	usernameMap := make(map[int64]string)
	for _, user := range users {
		usernameMap[user.ID] = user.Username
	}

	var result []*CompanyRelationWithSuperior
	for _, rel := range relations {
		result = append(result, &CompanyRelationWithSuperior{
			ID:                rel.ID,
			CompanyID:         rel.CompanyID,
			SuperiorUserID:    rel.SuperiorUserID,
			SubordinateUserID: rel.SubordinateUserID,
			ProjectID:         rel.ProjectID,
			Created:           rel.Created,
			Updated:           rel.Updated,
			SuperiorUsername:  usernameMap[rel.SuperiorUserID],
		})
	}

	return result, nil
}
