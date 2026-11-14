package seed

import (
	"fmt"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
)

func safePrefix(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// seedAssignments seeds 1 assignment per module (or retrieves existing ones) and returns them.
func seedAssignments(modules []*course.Module) ([]*content.Content, error) {
	var contentService content.IContentService
	app.Invoke(func(c content.IContentService) {
		contentService = c
	})

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
				Title:    fmt.Sprintf("Assignment 1 - %s Setup", safePrefix(mod.Title, 6)),
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
				Title:    fmt.Sprintf("Assignment 2 - %s Project Part 1", safePrefix(mod.Title, 6)),
				Position: 1,
			}
			asgn := &content.Assignment{
				Type:                   content.AssignmentHandIn,
				Deadline:               &pastDeadline,
				AcceptsLateSubmissions: true,
				AllowsGroupSubmission:  true,
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
