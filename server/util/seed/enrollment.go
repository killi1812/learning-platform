package seed

import (
	"errors"
	"fmt"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
)

// seedEnrollments enrolls teachers and students into courses and returns all created or existing enrollments.
func seedEnrollments(courses []*course.Course, teachers []*user.User, students []*user.User) ([]*course.Enrollment, error) {
	if len(courses) < 2 || len(teachers) < 2 || len(students) < 10 {
		return nil, errors.New("insufficient courses, teachers, or students for enrollments")
	}

	var courseService course.ICourseService
	app.Invoke(func(c course.ICourseService) {
		courseService = c
	})

	c1 := courses[0]
	c2 := courses[1]
	enrolled := make([]*course.Enrollment, 0, 18)

	enrollSafe := func(courseUuid uuid.UUID, email string, role course.CourseRole) (*course.Enrollment, error) {
		enrollments, err := courseService.GetCourseEnrollments(courseUuid)
		if err == nil {
			for i := range enrollments {
				if enrollments[i].UserEmail == email || (enrollments[i].User != nil && enrollments[i].User.Email == email) {
					return &enrollments[i], nil
				}
			}
		}

		res, err := courseService.EnrollByEmail(courseUuid, email, role)
		if err != nil {
			if errors.Is(err, course.ErrAlreadyEnrolled) {
				return nil, nil
			}
			return nil, err
		}
		return res, nil
	}

	// Enroll teacher2 in course 1
	e, err := enrollSafe(c1.Uuid, teachers[1].Email, course.RoleTeacher)
	if err != nil {
		return nil, fmt.Errorf("failed to enroll teacher2 in course 1: %w", err)
	}
	if e != nil {
		enrolled = append(enrolled, e)
	}

	// Enroll students 1..10 in course 1
	for _, st := range students {
		e, err := enrollSafe(c1.Uuid, st.Email, course.RoleStudent)
		if err != nil {
			return nil, fmt.Errorf("failed to enroll student %s in course 1: %w", st.Email, err)
		}
		if e != nil {
			enrolled = append(enrolled, e)
		}
	}

	// Enroll students 1..5 in course 2 (students 6..10 remain un-enrolled for testing)
	for i := range 5 {
		e, err := enrollSafe(c2.Uuid, students[i].Email, course.RoleStudent)
		if err != nil {
			return nil, fmt.Errorf("failed to enroll student %s in course 2: %w", students[i].Email, err)
		}
		if e != nil {
			enrolled = append(enrolled, e)
		}
	}

	return enrolled, nil
}
