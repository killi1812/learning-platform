package seed

import (
	"fmt"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
)

// seedAssignments seeds 1 assignment per module (or retrieves existing ones) and returns them.
func seedAssignments(contentService content.IContentService, modules []*course.Module) ([]*content.Content, error) {
	now := time.Now()
	openDeadline := now.Add(14 * 24 * time.Hour)
	pastDeadline := now.Add(-2 * 24 * time.Hour)

	seededContents := make([]*content.Content, 0, len(modules))

	for i, mod := range modules {
		existing, err := contentService.GetAssignmentByModule(mod.Uuid)
		if err == nil && existing != nil {
			seededContents = append(seededContents, existing)
			continue
		}

		if i%2 == 0 {
			cnt := &content.Content{
				Title:    fmt.Sprintf("Assignment 1 - %s Setup", mod.Title[:6]),
				Position: 1,
			}
			asgn := &content.Assignment{
				Type:                   content.AssignmentHandIn,
				Deadline:               &openDeadline,
				AcceptsLateSubmissions: false,
			}
			created, err := contentService.CreateAssignment(mod.Uuid, cnt, asgn)
			if err != nil {
				return nil, fmt.Errorf("failed to create assignment 1 for module %s: %w", mod.Title, err)
			}
			seededContents = append(seededContents, created)
		} else {
			cnt := &content.Content{
				Title:    fmt.Sprintf("Assignment 2 - %s Project Part 1", mod.Title[:6]),
				Position: 1,
			}
			asgn := &content.Assignment{
				Type:                   content.AssignmentHandIn,
				Deadline:               &pastDeadline,
				AcceptsLateSubmissions: true,
			}
			created, err := contentService.CreateAssignment(mod.Uuid, cnt, asgn)
			if err != nil {
				return nil, fmt.Errorf("failed to create assignment 2 for module %s: %w", mod.Title, err)
			}
			seededContents = append(seededContents, created)
		}
	}

	return seededContents, nil
}
