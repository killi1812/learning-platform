package seed

import (
	"fmt"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
)

// seedGroups seeds 2 groups per course and places initial enrolled students in Group 1.
func seedGroups(courses []*course.Course, students []*user.User) ([]*course.Group, error) {
	var groupService course.IGroupService
	app.Invoke(func(g course.IGroupService) {
		groupService = g
	})

	allGroups := make([]*course.Group, 0, len(courses)*2)

	for _, crs := range courses {
		existingGroups, err := groupService.ListCourseGroups(crs.Uuid)
		if err != nil {
			return nil, fmt.Errorf("failed to list groups for course %s: %w", crs.Title, err)
		}

		findGroup := func(name string) *course.Group {
			for i := range existingGroups {
				if existingGroups[i].Name == name {
					return &existingGroups[i]
				}
			}
			return nil
		}

		g1 := findGroup("Group 1")
		if g1 == nil {
			created, err := groupService.CreateGroup(crs.Uuid, "Group 1", 4)
			if err != nil {
				return nil, fmt.Errorf("failed to create Group 1 for course %s: %w", crs.Title, err)
			}
			g1 = created
		}
		allGroups = append(allGroups, g1)

		g2 := findGroup("Group 2")
		if g2 == nil {
			created, err := groupService.CreateGroup(crs.Uuid, "Group 2", 4)
			if err != nil {
				return nil, fmt.Errorf("failed to create Group 2 for course %s: %w", crs.Title, err)
			}
			g2 = created
		}
		allGroups = append(allGroups, g2)
	}

	// Idempotently join students[0] and students[1] into Group 1 of course 1
	if len(courses) > 0 && len(students) >= 2 && len(allGroups) > 0 {
		g1 := allGroups[0]
		_, _ = groupService.UserJoinGroup(g1.Uuid, students[0].Uuid)
		_, _ = groupService.UserJoinGroup(g1.Uuid, students[1].Uuid)
	}

	return allGroups, nil
}
