package seed

import (
	"bytes"
	"fmt"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"github.com/google/uuid"
)

// seedFiles seeds 1 simple text file material per module (or retrieves existing ones) and returns them.
func seedFiles(courses []*course.Course, modules []*course.Module) ([]*content.File, error) {
	var contentService content.IContentService
	app.Invoke(func(c content.IContentService) {
		contentService = c
	})

	courseMap := make(map[uint]uuid.UUID, len(courses))
	for _, crs := range courses {
		courseMap[crs.ID] = crs.Uuid
	}

	seededFiles := make([]*content.File, 0, len(modules))

	for _, mod := range modules {
		crsUuid, ok := courseMap[mod.CourseID]
		if !ok || crsUuid == uuid.Nil {
			continue
		}

		existing, err := contentService.ListFiles(crsUuid, mod.Uuid, uuid.Nil, true)
		if err == nil && len(existing) > 0 {
			seededFiles = append(seededFiles, &existing[0])
			continue
		}

		fileData := fmt.Sprintf(
			"Welcome to %s!\n\nThis is the introductory course material for %s.\nPlease review this text file to complete this module's reading requirement.",
			mod.Title,
			mod.Title,
		)
		fileBytes := []byte(fileData)
		reader := bytes.NewReader(fileBytes)

		created, err := contentService.UploadFile(
			crsUuid,
			mod.Uuid,
			fmt.Sprintf("%s - Reading Material", mod.Title),
			"reading-material.txt",
			int64(len(fileBytes)),
			"text/plain",
			reader,
			fmt.Sprintf("Introduction and overview document for %s.", mod.Title),
			"",
		)
		if err != nil {
			return nil, fmt.Errorf("failed to upload seed file for module %s: %w", mod.Title, err)
		}

		seededFiles = append(seededFiles, created)
	}

	return seededFiles, nil
}
