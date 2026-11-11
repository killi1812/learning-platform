package seed

import (
	"fmt"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
)

// seedModules seeds 2 modules per course (or retrieves existing ones) and returns them.
func seedModules(courses []*course.Course) ([]*course.Module, error) {
	var courseService course.ICourseService
	app.Invoke(func(c course.ICourseService) {
		courseService = c
	})

	allModules := make([]*course.Module, 0, len(courses)*2)

	for _, crs := range courses {
		existingMods, err := courseService.ListModules(crs.Uuid)
		if err != nil {
			return nil, fmt.Errorf("failed to list modules for course %s: %w", crs.Title, err)
		}

		findMod := func(title string) *course.Module {
			for i := range existingMods {
				if existingMods[i].Title == title {
					return &existingMods[i]
				}
			}
			return nil
		}

		title1 := fmt.Sprintf("Week 1 - %s Introduction", safePrefix(crs.Title, 6))
		mod1 := findMod(title1)
		if mod1 == nil {
			var err error
			mod1, err = courseService.CreateModule(crs.Uuid, &course.Module{
				Title:    title1,
				Position: 1,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create module 1 for course %s: %w", crs.Title, err)
			}
		}
		allModules = append(allModules, mod1)

		title2 := fmt.Sprintf("Week 2 - %s Deep Dive", safePrefix(crs.Title, 6))
		mod2 := findMod(title2)
		if mod2 == nil {
			var err error
			mod2, err = courseService.CreateModule(crs.Uuid, &course.Module{
				Title:     title2,
				Position:  2,
				DependsOn: &course.Module{Uuid: mod1.Uuid},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create module 2 for course %s: %w", crs.Title, err)
			}
		}
		allModules = append(allModules, mod2)
	}

	return allModules, nil
}
