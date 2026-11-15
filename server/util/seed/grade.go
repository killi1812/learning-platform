package seed

import (
	"fmt"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// seedGrades creates an initial graded submission and an ungraded submission for dev testing.
func seedGrades(assignments []*content.Content, teachers []*user.User, students []*user.User) error {
	var db *gorm.DB
	app.Invoke(func(d *gorm.DB) {
		db = d
	})

	if len(assignments) == 0 || len(teachers) == 0 || len(students) == 0 {
		return nil
	}

	// Pick an individual assignment
	var individualContent *content.Content
	for _, c := range assignments {
		if c.Assignment != nil && c.Assignment.Type == content.AssignmentHandIn && !c.Assignment.AllowsGroupSubmission {
			individualContent = c
			break
		}
	}
	if individualContent == nil {
		return nil
	}

	teacher := teachers[0]
	student1 := students[0]

	now := time.Now()
	// 1. Graded submission for student 1
	var existingSub1 submission.IndividualSubmission
	err := db.Where("content_id = ? AND user_id = ?", individualContent.ID, student1.ID).First(&existingSub1).Error
	if err == gorm.ErrRecordNotFound {
		sub1 := submission.IndividualSubmission{
			Submission: submission.Submission{
				Uuid:        uuid.New(),
				ContentID:   individualContent.ID,
				Attempt:     1,
				IsLate:      false,
				PayloadType: submission.PayloadTypeURL,
				PayloadURL:  "https://git.app.uib.no/student/hw1-setup.git",
				SubmittedAt: now.Add(-2 * time.Hour),
			},
			UserID:    &student1.ID,
			UserEmail: student1.Email,
		}
		if err := db.Create(&sub1).Error; err != nil {
			return fmt.Errorf("failed to seed submission 1: %w", err)
		}
		existingSub1 = sub1
	}

	if existingSub1.ID != 0 {
		var existingGrade submission.Grade
		err := db.Where("individual_submission_id = ?", existingSub1.ID).First(&existingGrade).Error
		if err == gorm.ErrRecordNotFound {
			releasedAt := now.Add(-1 * time.Hour)
			feedback := "### Review Summary\n\n- **Architecture**: Clean modular structure.\n- **Tests**: High coverage across modules.\n\n```go\n// Minor suggestion: wrap errors with %w\nreturn fmt.Errorf(\"failed: %w\", err)\n```\n\nWell done!"
			grade := submission.Grade{
				IndividualSubmissionID: &existingSub1.ID,
				Score:                  92.5,
				Feedback:               feedback,
				IsVisibleToStudent:     true,
				ReleasedAt:             &releasedAt,
				GradedByID:             &teacher.ID,
			}
			if err := db.Create(&grade).Error; err != nil {
				return fmt.Errorf("failed to seed grade: %w", err)
			}
		}
	}

	// 2. Ungraded (pending) submission for student 2 if available
	if len(students) > 1 {
		student2 := students[1]
		var existingSub2 submission.IndividualSubmission
		err := db.Where("content_id = ? AND user_id = ?", individualContent.ID, student2.ID).First(&existingSub2).Error
		if err == gorm.ErrRecordNotFound {
			sub2 := submission.IndividualSubmission{
				Submission: submission.Submission{
					Uuid:        uuid.New(),
					ContentID:   individualContent.ID,
					Attempt:     1,
					IsLate:      false,
					PayloadType: submission.PayloadTypeURL,
					PayloadURL:  "https://git.app.uib.no/student2/hw1-setup.git",
					SubmittedAt: now.Add(-30 * time.Minute),
				},
				UserID:    &student2.ID,
				UserEmail: student2.Email,
			}
			if err := db.Create(&sub2).Error; err != nil {
				return fmt.Errorf("failed to seed submission 2: %w", err)
			}
		}
	}

	// 3. Quiz attempt for student 1 if a quiz assignment exists
	for _, c := range assignments {
		if c.Assignment != nil && c.Assignment.Type == content.AssignmentQuiz {
			type attemptRow struct {
				ID uint
			}
			var existingAttempt attemptRow
			qErr := db.Table("attempts").Where("user_id = ? AND quiz_id = ?", student1.ID, c.ID).First(&existingAttempt).Error
			if qErr == gorm.ErrRecordNotFound {
				submittedTime := now.Add(-90 * time.Minute)
				rawAttempt := map[string]interface{}{
					"user_id":      student1.ID,
					"quiz_id":      c.ID,
					"started_at":   now.Add(-100 * time.Minute),
					"submitted_at": submittedTime,
					"percent":      88.0,
					"passed":       true,
					"created_at":   now,
					"updated_at":   now,
				}
				if err := db.Table("attempts").Create(rawAttempt).Error; err != nil {
					return fmt.Errorf("failed to seed quiz attempt: %w", err)
				}
			}
			break
		}
	}

	return nil
}
