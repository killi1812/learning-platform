package seed

import (
	"fmt"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
)

// seedStudents creates 10 student accounts (or retrieves existing ones) and returns them.
func seedStudents() ([]*user.User, error) {
	var userCrud user.IUserCrudService
	app.Invoke(func(u user.IUserCrudService) {
		userCrud = u
	})

	students := make([]*user.User, 10)
	for i := range 10 {
		num := i + 1
		username := fmt.Sprintf("student%d", num)
		email := fmt.Sprintf("student%d@example.com", num)

		// 1. Check if user already exists by username
		existing, err := userCrud.GetUserByUsername(username)
		if err == nil && existing != nil {
			if existing.Email != email {
				existing.Email = email
				_, _ = userCrud.Update(existing.Uuid, existing)
			}
			students[i] = existing
			continue
		}

		// 2. Check if user already exists by email
		existing, err = userCrud.GetUserByEmail(email)
		if err == nil && existing != nil {
			students[i] = existing
			continue
		}

		// 3. Otherwise create new student
		u := &user.User{
			Username: username,
			Email:    email,
			Role:     authutils.ROLE_USER,
		}
		created, err := userCrud.Create(u, _DEFAULT_SEED_PASSWORD)
		if err != nil {
			// Never fail if user is already in the database: fetch by username or email
			if found, fetchErr := userCrud.GetUserByUsername(username); fetchErr == nil && found != nil {
				students[i] = found
				continue
			}
			if found, fetchErr := userCrud.GetUserByEmail(email); fetchErr == nil && found != nil {
				students[i] = found
				continue
			}
			return nil, fmt.Errorf("failed to create seed student %d: %w", num, err)
		}
		students[i] = created
	}
	return students, nil
}
