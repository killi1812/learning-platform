package seed

import (
	"fmt"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
)

const _DEFAULT_SEED_PASSWORD = "Pa$$w0rd"

// seedTeachers creates 3 teacher accounts (or retrieves existing ones) and returns them.
func seedTeachers() ([]*user.User, error) {
	var userCrud user.IUserCrudService
	app.Invoke(func(u user.IUserCrudService) {
		userCrud = u
	})

	teachers := make([]*user.User, 3)
	for i := range 3 {
		num := i + 1
		username := fmt.Sprintf("teacher%d", num)
		email := fmt.Sprintf("teacher%d@example.com", num)

		// 1. Check if user already exists by username
		existing, err := userCrud.GetUserByUsername(username)
		if err == nil && existing != nil {
			if existing.Email != email {
				existing.Email = email
				_, _ = userCrud.Update(existing.Uuid, existing)
			}
			teachers[i] = existing
			continue
		}

		// 2. Check if user already exists by email
		existing, err = userCrud.GetUserByEmail(email)
		if err == nil && existing != nil {
			teachers[i] = existing
			continue
		}

		// 3. Otherwise create new teacher
		u := &user.User{
			Username: username,
			Email:    email,
			Role:     authutils.ROLE_USER,
		}
		created, err := userCrud.Create(u, _DEFAULT_SEED_PASSWORD)
		if err != nil {
			// Never fail if user is already in the database: fetch by username or email
			if found, fetchErr := userCrud.GetUserByUsername(username); fetchErr == nil && found != nil {
				teachers[i] = found
				continue
			}
			if found, fetchErr := userCrud.GetUserByEmail(email); fetchErr == nil && found != nil {
				teachers[i] = found
				continue
			}
			return nil, fmt.Errorf("failed to create seed teacher %d: %w", num, err)
		}
		teachers[i] = created
	}
	return teachers, nil
}
