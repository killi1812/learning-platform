package seed

import (
	"errors"
	"fmt"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
)

// seedCourses creates 2 seed courses (or retrieves existing ones) using the provided teachers and returns them.
func seedCourses(courseService course.ICourseService, teachers []*user.User) ([]*course.Course, error) {
	if len(teachers) < 3 {
		return nil, errors.New("seedCourses requires at least 3 teachers")
	}

	existingCourses, err := courseService.ListCourses(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing courses: %w", err)
	}

	findCourse := func(title string, list []course.Course) *course.Course {
		for i := range list {
			if list[i].Title == title {
				return &list[i]
			}
		}
		return nil
	}

	courses := make([]*course.Course, 2)

	// Course 1
	if c := findCourse("DAT250 Software Technology", existingCourses); c != nil {
		courses[0] = c
	} else {
		c1 := &course.Course{
			Title:       "DAT250 Software Technology",
			Description: "Study of advanced software engineering principles, microservices, and messaging.",
			Term:        course.TermAutumn,
			Year:        2026,
		}
		created, err := courseService.CreateCourse(c1, teachers[0].Uuid)
		if err != nil {
			if latest, listErr := courseService.ListCourses(nil, nil); listErr == nil {
				courses[0] = findCourse(c1.Title, latest)
			}
			if courses[0] == nil {
				return nil, fmt.Errorf("failed to create seed course 1: %w", err)
			}
		} else {
			courses[0] = created
		}
	}

	// Course 2
	if c := findCourse("DAT251 Modern Software Development", existingCourses); c != nil {
		courses[1] = c
	} else {
		c2 := &course.Course{
			Title:       "DAT251 Modern Software Development",
			Description: "Applied software engineering, agile practices, and automated pipelines.",
			Term:        course.TermSpring,
			Year:        2027,
		}
		created, err := courseService.CreateCourse(c2, teachers[2].Uuid)
		if err != nil {
			if latest, listErr := courseService.ListCourses(nil, nil); listErr == nil {
				courses[1] = findCourse(c2.Title, latest)
			}
			if courses[1] == nil {
				return nil, fmt.Errorf("failed to create seed course 2: %w", err)
			}
		} else {
			courses[1] = created
		}
	}

	return courses, nil
}
