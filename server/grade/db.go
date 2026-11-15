package grade

import (
	"errors"
	"fmt"
	"io"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/content"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/course"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/notification"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/quiz"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/submission"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrSubmissionNotFound       = errors.New("submission not found")
	ErrScoreOutOfRange          = errors.New("score exceeds assignment max points")
	ErrNotHandin                = errors.New("only hand-in assignments can be graded manually")
	ErrGroupGradingNotSupported = errors.New("group assignment grading is not supported yet")
	ErrContentNotFound          = errors.New("assignment not found")
	ErrGraderNotFound           = errors.New("grader user not found")
)

type IGradeService interface {
	UpsertGrade(courseUuid, contentUuid, submissionUuid, graderUuid uuid.UUID, dto UpsertGradeDto) (*GradeDto, error)
	ReleaseAssignmentGrades(courseUuid, contentUuid uuid.UUID) (int, error)
	ListPending(userUuid uuid.UUID, isAdmin bool, courseUuid *uuid.UUID) ([]PendingSubmissionDto, error)
	ListMyGrades(userUuid uuid.UUID, courseUuid *uuid.UUID) ([]StudentGradeDto, error)
	ExportCourseCSV(courseUuid uuid.UUID, w io.Writer) error
}

type GradeService struct {
	db            *gorm.DB
	notifications notification.INotifier
	logger        *zap.SugaredLogger
}

func NewGradeService(db *gorm.DB, notifications notification.INotifier, logger *zap.SugaredLogger) IGradeService {
	return &GradeService{
		db:            db,
		notifications: notifications,
		logger:        logger,
	}
}

// UpsertGrade creates or updates a manual grade for an individual submission.
func (s *GradeService) UpsertGrade(courseUuid, contentUuid, submissionUuid, graderUuid uuid.UUID, dto UpsertGradeDto) (*GradeDto, error) {
	var grader user.User
	if err := s.db.Where("uuid = ?", graderUuid).First(&grader).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGraderNotFound
		}
		return nil, err
	}

	var cnt content.Content
	err := s.db.
		Joins("JOIN modules ON modules.id = contents.module_id").
		Joins("JOIN courses ON courses.id = modules.course_id").
		Preload("Assignment").
		Where("contents.uuid = ? AND courses.uuid = ?", contentUuid, courseUuid).
		First(&cnt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContentNotFound
		}
		return nil, err
	}

	if cnt.Assignment == nil || cnt.Assignment.Type != content.AssignmentHandIn {
		return nil, ErrNotHandin
	}
	if cnt.Assignment.AllowsGroupSubmission {
		return nil, ErrGroupGradingNotSupported
	}

	var indSub submission.IndividualSubmission
	err = s.db.
		Where("uuid = ? AND content_id = ?", submissionUuid, cnt.ID).
		First(&indSub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Check if target is a group submission to return precise error
			var grpSub submission.GroupSubmission
			if gErr := s.db.Where("uuid = ? AND content_id = ?", submissionUuid, cnt.ID).First(&grpSub).Error; gErr == nil {
				return nil, ErrGroupGradingNotSupported
			}
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}

	maxPoints := cnt.Assignment.MaxPoints
	if maxPoints <= 0 {
		maxPoints = 100
	}
	if *dto.Score > maxPoints {
		return nil, ErrScoreOutOfRange
	}

	var studentUuid uuid.UUID
	if indSub.UserID != nil {
		var student user.User
		if err := s.db.Select("uuid").Where("id = ?", *indSub.UserID).First(&student).Error; err == nil {
			studentUuid = student.Uuid
		}
	}

	now := time.Now()
	var shouldNotify bool
	var savedGrade submission.Grade

	err = s.db.Transaction(func(tx *gorm.DB) error {
		var existing submission.Grade
		findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("individual_submission_id = ?", indSub.ID).
			First(&existing).Error

		if findErr == nil {
			wasVisible := existing.IsVisibleToStudent
			existing.Score = *dto.Score
			existing.Feedback = dto.Feedback
			existing.IsVisibleToStudent = dto.IsVisibleToStudent
			existing.GradedByID = &grader.ID

			if !wasVisible && dto.IsVisibleToStudent {
				existing.ReleasedAt = &now
				shouldNotify = true
			} else if !dto.IsVisibleToStudent {
				existing.ReleasedAt = nil
			}

			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
			savedGrade = existing
			return nil
		} else if errors.Is(findErr, gorm.ErrRecordNotFound) {
			var relAt *time.Time
			if dto.IsVisibleToStudent {
				relAt = &now
				shouldNotify = true
			}
			newGrade := submission.Grade{
				IndividualSubmissionID: &indSub.ID,
				Score:                  *dto.Score,
				Feedback:               dto.Feedback,
				IsVisibleToStudent:     dto.IsVisibleToStudent,
				ReleasedAt:             relAt,
				GradedByID:             &grader.ID,
			}
			if err := tx.Create(&newGrade).Error; err != nil {
				return err
			}
			savedGrade = newGrade
			return nil
		}
		return findErr
	})

	if err != nil {
		s.logger.Errorf("Failed to save grade for submission %s: %v", submissionUuid, err)
		return nil, err
	}

	if shouldNotify && studentUuid != uuid.Nil {
		_, nErr := s.notifications.NotifySystem(studentUuid, notification.NotificationContent{
			Title: fmt.Sprintf("Grade Published: %s", cnt.Title),
			Body:  fmt.Sprintf("Your submission for %q has been graded (%g/%g).", cnt.Title, savedGrade.Score, maxPoints),
			URL:   "/grades",
		})
		if nErr != nil {
			s.logger.Warnf("Failed to notify student %s of grade release: %v", studentUuid, nErr)
		}
	}

	return &GradeDto{
		Score:              savedGrade.Score,
		MaxPoints:          maxPoints,
		Feedback:           savedGrade.Feedback,
		IsVisibleToStudent: savedGrade.IsVisibleToStudent,
		GradedByEmail:      grader.Email,
		GradedAt:           savedGrade.UpdatedAt,
		ReleasedAt:         savedGrade.ReleasedAt,
	}, nil
}

// ReleaseAssignmentGrades releases all unreleased grades for an individual assignment.
func (s *GradeService) ReleaseAssignmentGrades(courseUuid, contentUuid uuid.UUID) (int, error) {
	var cnt content.Content
	err := s.db.
		Joins("JOIN modules ON modules.id = contents.module_id").
		Joins("JOIN courses ON courses.id = modules.course_id").
		Preload("Assignment").
		Where("contents.uuid = ? AND courses.uuid = ?", contentUuid, courseUuid).
		First(&cnt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrContentNotFound
		}
		return 0, err
	}

	if cnt.Assignment == nil || cnt.Assignment.Type != content.AssignmentHandIn {
		return 0, ErrNotHandin
	}
	if cnt.Assignment.AllowsGroupSubmission {
		return 0, ErrGroupGradingNotSupported
	}

	type unreleasedRow struct {
		GradeID uint      `gorm:"column:grade_id"`
		UserID  uint      `gorm:"column:user_id"`
		UserUid uuid.UUID `gorm:"column:user_uuid"`
	}

	var rows []unreleasedRow
	err = s.db.Table("grades").
		Select("grades.id as grade_id, users.id as user_id, users.uuid as user_uuid").
		Joins("JOIN individual_submissions ON individual_submissions.id = grades.individual_submission_id").
		Joins("JOIN users ON users.id = individual_submissions.user_id").
		Where("individual_submissions.content_id = ? AND grades.is_visible_to_student = false AND grades.deleted_at IS NULL", cnt.ID).
		Scan(&rows).Error
	if err != nil {
		return 0, err
	}

	if len(rows) == 0 {
		return 0, nil
	}

	gradeIDs := make([]uint, 0, len(rows))
	userUuidMap := make(map[uuid.UUID]bool, len(rows))
	for _, r := range rows {
		gradeIDs = append(gradeIDs, r.GradeID)
		if r.UserUid != uuid.Nil {
			userUuidMap[r.UserUid] = true
		}
	}

	now := time.Now()
	err = s.db.Model(&submission.Grade{}).
		Where("id IN ?", gradeIDs).
		Updates(map[string]any{
			"is_visible_to_student": true,
			"released_at":           now,
		}).Error
	if err != nil {
		s.logger.Errorf("Failed to release grades for assignment %s: %v", contentUuid, err)
		return 0, err
	}

	if len(userUuidMap) > 0 {
		userUuids := make([]uuid.UUID, 0, len(userUuidMap))
		for u := range userUuidMap {
			userUuids = append(userUuids, u)
		}
		_, nErr := s.notifications.NotifySystemMany(userUuids, notification.NotificationContent{
			Title: fmt.Sprintf("Grades Published: %s", cnt.Title),
			Body:  fmt.Sprintf("Grades for %q have been published.", cnt.Title),
			URL:   "/grades",
		})
		if nErr != nil {
			s.logger.Warnf("Failed to send bulk grade release notifications: %v", nErr)
		}
	}

	return len(gradeIDs), nil
}

// ListPending returns latest ungraded individual submissions across courses taught by the user.
func (s *GradeService) ListPending(userUuid uuid.UUID, isAdmin bool, courseUuid *uuid.UUID) ([]PendingSubmissionDto, error) {
	var u user.User
	if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		return nil, err
	}

	var courseIDs []uint
	if !isAdmin {
		err := s.db.Model(&course.Enrollment{}).
			Where("user_id = ? AND role = ? AND deleted_at IS NULL", u.ID, course.RoleTeacher).
			Pluck("course_id", &courseIDs).Error
		if err != nil {
			return nil, err
		}
		if len(courseIDs) == 0 {
			return []PendingSubmissionDto{}, nil
		}
	}

	type queryResult struct {
		SubmissionUuid  uuid.UUID  `gorm:"column:submission_uuid"`
		CourseUuid      uuid.UUID  `gorm:"column:course_uuid"`
		ContentUuid     uuid.UUID  `gorm:"column:content_uuid"`
		CourseTitle     string     `gorm:"column:course_title"`
		CourseTerm      string     `gorm:"column:course_term"`
		CourseYear      int        `gorm:"column:course_year"`
		AssignmentTitle string     `gorm:"column:assignment_title"`
		UserEmail       string     `gorm:"column:user_email"`
		UserName        string     `gorm:"column:user_name"`
		Attempt         int        `gorm:"column:attempt"`
		IsLate          bool       `gorm:"column:is_late"`
		SubmittedAt     time.Time  `gorm:"column:submitted_at"`
		Deadline        *time.Time `gorm:"column:deadline"`
		MaxPoints       float64    `gorm:"column:max_points"`
	}

	query := s.db.Table("individual_submissions s").
		Select(`s.uuid as submission_uuid,
			crs.uuid as course_uuid,
			cnt.uuid as content_uuid,
			crs.title as course_title,
			crs.term as course_term,
			crs.year as course_year,
			cnt.title as assignment_title,
			s.user_email as user_email,
			COALESCE(u.username, '') as user_name,
			s.attempt as attempt,
			s.is_late as is_late,
			s.submitted_at as submitted_at,
			asgn.deadline as deadline,
			COALESCE(asgn.max_points, 100) as max_points`).
		Joins("JOIN contents cnt ON cnt.id = s.content_id AND cnt.deleted_at IS NULL").
		Joins("JOIN assignments asgn ON asgn.content_id = cnt.id").
		Joins("JOIN modules mod ON mod.id = cnt.module_id AND mod.deleted_at IS NULL").
		Joins("JOIN courses crs ON crs.id = mod.course_id AND crs.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = s.user_id").
		Joins("LEFT JOIN grades g ON g.individual_submission_id = s.id AND g.deleted_at IS NULL").
		Where("asgn.type = ? AND asgn.allows_group_submission = false", content.AssignmentHandIn).
		Where("g.id IS NULL").
		Where("s.deleted_at IS NULL").
		Where("NOT EXISTS (SELECT 1 FROM individual_submissions s2 WHERE s2.user_id = s.user_id AND s2.content_id = s.content_id AND s2.attempt > s.attempt AND s2.deleted_at IS NULL)")

	if !isAdmin {
		query = query.Where("crs.id IN ?", courseIDs)
	}
	if courseUuid != nil {
		query = query.Where("crs.uuid = ?", *courseUuid)
	}

	var results []queryResult
	err := query.Order("s.submitted_at ASC").Limit(500).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	dtos := make([]PendingSubmissionDto, 0, len(results))
	for _, r := range results {
		courseCode := fmt.Sprintf("%s %d", r.CourseTerm, r.CourseYear)
		maxPts := r.MaxPoints
		if maxPts <= 0 {
			maxPts = 100
		}
		dtos = append(dtos, PendingSubmissionDto{
			SubmissionUuid:  r.SubmissionUuid,
			CourseUuid:      r.CourseUuid,
			ContentUuid:     r.ContentUuid,
			CourseCode:      courseCode,
			CourseName:      r.CourseTitle,
			AssignmentTitle: r.AssignmentTitle,
			UserEmail:       r.UserEmail,
			UserName:        r.UserName,
			Attempt:         r.Attempt,
			IsLate:          r.IsLate,
			SubmittedAt:     r.SubmittedAt,
			Deadline:        r.Deadline,
			MaxPoints:       maxPts,
		})
	}

	return dtos, nil
}

// ListMyGrades returns all released hand-in grades and quiz best-attempt scores for the student.
func (s *GradeService) ListMyGrades(userUuid uuid.UUID, courseUuid *uuid.UUID) ([]StudentGradeDto, error) {
	var u user.User
	if err := s.db.Where("uuid = ?", userUuid).First(&u).Error; err != nil {
		return nil, err
	}

	// 1. Fetch released hand-in grades
	type handinGradeRow struct {
		CourseUuid      uuid.UUID `gorm:"column:course_uuid"`
		ContentUuid     uuid.UUID `gorm:"column:content_uuid"`
		CourseTitle     string    `gorm:"column:course_title"`
		CourseTerm      string    `gorm:"column:course_term"`
		CourseYear      int       `gorm:"column:course_year"`
		ModuleTitle     string    `gorm:"column:module_title"`
		AssignmentTitle string    `gorm:"column:assignment_title"`
		Attempt         int       `gorm:"column:attempt"`
		Score           float64   `gorm:"column:score"`
		MaxPoints       float64   `gorm:"column:max_points"`
		Feedback        string    `gorm:"column:feedback"`
		GradedAt        time.Time `gorm:"column:graded_at"`
	}

	hQuery := s.db.Table("grades g").
		Select(`crs.uuid as course_uuid,
			cnt.uuid as content_uuid,
			crs.title as course_title,
			crs.term as course_term,
			crs.year as course_year,
			mod.title as module_title,
			cnt.title as assignment_title,
			ind.attempt as attempt,
			g.score as score,
			COALESCE(asgn.max_points, 100) as max_points,
			g.feedback as feedback,
			g.updated_at as graded_at`).
		Joins("JOIN individual_submissions ind ON ind.id = g.individual_submission_id AND ind.deleted_at IS NULL").
		Joins("JOIN contents cnt ON cnt.id = ind.content_id AND cnt.deleted_at IS NULL").
		Joins("JOIN assignments asgn ON asgn.content_id = cnt.id").
		Joins("JOIN modules mod ON mod.id = cnt.module_id AND mod.deleted_at IS NULL").
		Joins("JOIN courses crs ON crs.id = mod.course_id AND crs.deleted_at IS NULL").
		Where("ind.user_id = ? AND g.is_visible_to_student = ? AND g.deleted_at IS NULL", u.ID, true)

	if courseUuid != nil {
		hQuery = hQuery.Where("crs.uuid = ?", *courseUuid)
	}

	var hRows []handinGradeRow
	if err := hQuery.Order("crs.title ASC, mod.position ASC, cnt.position ASC").Scan(&hRows).Error; err != nil {
		return nil, err
	}

	dtos := make([]StudentGradeDto, 0, len(hRows))
	for _, r := range hRows {
		maxPts := r.MaxPoints
		if maxPts <= 0 {
			maxPts = 100
		}
		pct := 0.0
		if maxPts > 0 {
			pct = (r.Score / maxPts) * 100.0
		}
		dtos = append(dtos, StudentGradeDto{
			Type:            GradeTypeHandin,
			CourseUuid:      r.CourseUuid,
			ContentUuid:     r.ContentUuid,
			CourseName:      r.CourseTitle,
			CourseCode:      fmt.Sprintf("%s %d", r.CourseTerm, r.CourseYear),
			ModuleTitle:     r.ModuleTitle,
			AssignmentTitle: r.AssignmentTitle,
			Attempt:         r.Attempt,
			Score:           r.Score,
			MaxPoints:       maxPts,
			Percent:         pct,
			Feedback:        r.Feedback,
			GradedAt:        r.GradedAt,
		})
	}

	// 2. Fetch published quiz best attempts
	type quizRow struct {
		ContentID       uint      `gorm:"column:content_id"`
		ContentUuid     uuid.UUID `gorm:"column:content_uuid"`
		CourseUuid      uuid.UUID `gorm:"column:course_uuid"`
		CourseTitle     string    `gorm:"column:course_title"`
		CourseTerm      string    `gorm:"column:course_term"`
		CourseYear      int       `gorm:"column:course_year"`
		ModuleTitle     string    `gorm:"column:module_title"`
		AssignmentTitle string    `gorm:"column:assignment_title"`
	}

	qQuery := s.db.Table("contents cnt").
		Select(`cnt.id as content_id,
			cnt.uuid as content_uuid,
			crs.uuid as course_uuid,
			crs.title as course_title,
			crs.term as course_term,
			crs.year as course_year,
			mod.title as module_title,
			cnt.title as assignment_title`).
		Joins("JOIN quizzes q ON q.content_id = cnt.id").
		Joins("JOIN modules mod ON mod.id = cnt.module_id AND mod.deleted_at IS NULL").
		Joins("JOIN courses crs ON crs.id = mod.course_id AND crs.deleted_at IS NULL").
		Joins("JOIN enrollments enr ON enr.course_id = crs.id AND enr.user_id = ? AND enr.deleted_at IS NULL", u.ID).
		Where("cnt.deleted_at IS NULL AND q.published = ?", true)

	if courseUuid != nil {
		qQuery = qQuery.Where("crs.uuid = ?", *courseUuid)
	}

	var qRows []quizRow
	if err := qQuery.Order("crs.title ASC, mod.position ASC, cnt.position ASC").Scan(&qRows).Error; err != nil {
		return nil, err
	}

	for _, qr := range qRows {
		var attempts []quiz.Attempt
		err := s.db.Where("user_id = ? AND quiz_id = ? AND submitted_at IS NOT NULL", u.ID, qr.ContentID).
			Order("percent DESC, id DESC").
			Find(&attempts).Error
		if err != nil || len(attempts) == 0 {
			continue
		}

		best := attempts[0]
		submittedTime := time.Now()
		if best.SubmittedAt != nil {
			submittedTime = *best.SubmittedAt
		}

		dtos = append(dtos, StudentGradeDto{
			Type:            GradeTypeQuiz,
			CourseUuid:      qr.CourseUuid,
			ContentUuid:     qr.ContentUuid,
			CourseName:      qr.CourseTitle,
			CourseCode:      fmt.Sprintf("%s %d", qr.CourseTerm, qr.CourseYear),
			ModuleTitle:     qr.ModuleTitle,
			AssignmentTitle: qr.AssignmentTitle,
			Attempt:         len(attempts),
			Score:           best.Percent,
			MaxPoints:       100,
			Percent:         best.Percent,
			Feedback:        "",
			GradedAt:        submittedTime,
		})
	}

	return dtos, nil
}

// ExportCourseCSV exports the complete gradebook matrix for all students in a course.
func (s *GradeService) ExportCourseCSV(courseUuid uuid.UUID, w io.Writer) error {
	var crs course.Course
	if err := s.db.Where("uuid = ?", courseUuid).First(&crs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return course.ErrCourseNotFound
		}
		return err
	}

	var enrollments []course.Enrollment
	err := s.db.Preload("User").
		Where("course_id = ? AND role = ? AND deleted_at IS NULL", crs.ID, course.RoleStudent).
		Order("user_email ASC").
		Find(&enrollments).Error
	if err != nil {
		return err
	}

	var contents []content.Content
	err = s.db.
		Joins("JOIN modules ON modules.id = contents.module_id AND modules.deleted_at IS NULL").
		Preload("Module").
		Preload("Assignment").
		Preload("Assignment.Quiz").
		Where("modules.course_id = ? AND contents.type = ? AND contents.deleted_at IS NULL", crs.ID, content.TypeAssignment).
		Order("modules.position ASC, contents.position ASC").
		Find(&contents).Error
	if err != nil {
		return err
	}

	records := make([]GradebookRecord, 0, len(enrollments)*len(contents))

	for _, enr := range enrollments {
		stUser := enr.User
		if stUser == nil {
			continue
		}

		for _, cnt := range contents {
			modTitle := ""
			if cnt.Module != nil {
				modTitle = cnt.Module.Title
			}

			asgn := cnt.Assignment
			if asgn == nil {
				continue
			}

			rec := GradebookRecord{
				StudentEmail:    enr.UserEmail,
				StudentName:     stUser.Username,
				ModuleTitle:     modTitle,
				AssignmentTitle: cnt.Title,
			}

			if asgn.Type == content.AssignmentQuiz {
				rec.AssignmentType = "QUIZ"
				var attempts []quiz.Attempt
				_ = s.db.Where("user_id = ? AND quiz_id = ? AND submitted_at IS NOT NULL", stUser.ID, cnt.ID).
					Order("percent DESC, id DESC").
					Find(&attempts).Error

				if len(attempts) > 0 {
					best := attempts[0]
					rec.Attempt = fmt.Sprintf("%d", len(attempts))
					if best.SubmittedAt != nil {
						rec.SubmittedAt = best.SubmittedAt.Format(time.RFC3339)
					}
					rec.Score = fmt.Sprintf("%.2f", best.Percent)
					rec.MaxPoints = "100.00"
					rec.Percent = fmt.Sprintf("%.2f%%", best.Percent)
					rec.IsVisibleToStudent = "true"
					rec.GradedAt = rec.SubmittedAt
				}
				records = append(records, rec)
				continue
			}

			if asgn.AllowsGroupSubmission {
				rec.AssignmentType = "HANDIN (GROUP)"
				rec.Score = "N/A (Group)"
				rec.MaxPoints = fmt.Sprintf("%.2f", asgn.MaxPoints)
				rec.Percent = "N/A"
				rec.IsVisibleToStudent = "false"
				records = append(records, rec)
				continue
			}

			rec.AssignmentType = "HANDIN"
			maxPts := asgn.MaxPoints
			if maxPts <= 0 {
				maxPts = 100
			}
			rec.MaxPoints = fmt.Sprintf("%.2f", maxPts)

			var subs []submission.IndividualSubmission
			_ = s.db.Preload("Grade").
				Preload("Grade.GradedBy").
				Where("user_id = ? AND content_id = ?", stUser.ID, cnt.ID).
				Order("attempt DESC").
				Find(&subs).Error

			if len(subs) > 0 {
				var targetSub *submission.IndividualSubmission
				for i := range subs {
					if subs[i].Grade != nil {
						targetSub = &subs[i]
						break
					}
				}
				if targetSub == nil {
					targetSub = &subs[0]
				}

				rec.Attempt = fmt.Sprintf("%d", targetSub.Attempt)
				rec.SubmittedAt = targetSub.SubmittedAt.Format(time.RFC3339)
				if targetSub.IsLate {
					rec.IsLate = "true"
				} else {
					rec.IsLate = "false"
				}

				if targetSub.Grade != nil {
					gr := targetSub.Grade
					rec.Score = fmt.Sprintf("%.2f", gr.Score)
					pct := 0.0
					if maxPts > 0 {
						pct = (gr.Score / maxPts) * 100.0
					}
					rec.Percent = fmt.Sprintf("%.2f%%", pct)
					if gr.IsVisibleToStudent {
						rec.IsVisibleToStudent = "true"
					} else {
						rec.IsVisibleToStudent = "false"
					}
					if gr.GradedBy != nil {
						rec.GradedByEmail = gr.GradedBy.Email
					}
					rec.GradedAt = gr.UpdatedAt.Format(time.RFC3339)
					rec.Feedback = gr.Feedback
				}
			}

			records = append(records, rec)
		}
	}

	return WriteGradebookCSV(w, records)
}
