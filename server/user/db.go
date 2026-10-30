package user

import (
	"errors"
	"sort"
	"strings"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"

	"github.com/google/uuid"
	"github.com/xrash/smetrics"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ErrRecordNotFound = errors.New("record not found")

type IUserCrudService interface {
	Create(user *User, password string) (*User, error)
	Read(uuid uuid.UUID) (*User, error)
	ReadAll() ([]User, error)
	Update(uuid uuid.UUID, user *User) (*User, error)
	Delete(uuid uuid.UUID) error
	GetAllUsers() ([]User, error)
	SearchUsersByName(query string) ([]User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByUsername(username string) (*User, error)
}

type UserCrudService struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

type UserWithScore struct {
	User  User
	Score float64
}

func NewUserCrudService(db *gorm.DB, logger *zap.SugaredLogger) IUserCrudService {
	return &UserCrudService{
		db:     db,
		logger: logger,
	}
}

func (u *UserCrudService) ReadAll() ([]User, error) {
	var users []User
	if err := u.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (u *UserCrudService) Delete(userUuid uuid.UUID) error {
	existing, err := u.Read(userUuid)
	if err != nil {
		return err
	}

	if err := u.db.Delete(existing).Error; err != nil {
		return err
	}
	return nil
}

func (u *UserCrudService) Read(userUuid uuid.UUID) (*User, error) {
	var user User
	err := u.db.Where("uuid = ?", userUuid).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (u *UserCrudService) Update(userUuid uuid.UUID, user *User) (*User, error) {
	userOld, err := u.Read(userUuid)
	if err != nil {
		return nil, err
	}

	userOld = userOld.Update(user)

	if err := u.db.Save(userOld).Error; err != nil {
		return nil, err
	}
	return userOld, nil
}

func (u *UserCrudService) Create(user *User, password string) (*User, error) {
	var count int64
	if err := u.db.Model(&User{}).Where("username = ?", user.Username).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("username already exists")
	}

	hash, err := authutils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user.PasswordHash = hash
	if user.Uuid == uuid.Nil {
		user.Uuid = uuid.New()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = time.Now()
	}

	if err := u.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (u *UserCrudService) GetAllUsers() ([]User, error) {
	return u.ReadAll()
}

func (u *UserCrudService) SearchUsersByName(query string) ([]User, error) {
	users, err := u.ReadAll()
	if err != nil {
		return nil, err
	}

	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	var scoredUsers []UserWithScore
	for _, user := range users {
		fullName := strings.ToLower(user.Username + " " + user.Email)
		score := smetrics.JaroWinkler(normalizedQuery, fullName, 0.7, 4)
		scoredUsers = append(scoredUsers, UserWithScore{User: user, Score: score})
	}

	sort.Slice(scoredUsers, func(i, j int) bool {
		return scoredUsers[i].Score > scoredUsers[j].Score
	})

	var filteredUsers []User
	for _, scoredUser := range scoredUsers {
		if scoredUser.Score >= 0.8 {
			filteredUsers = append(filteredUsers, scoredUser.User)
		}
	}

	return filteredUsers, nil
}

func (u *UserCrudService) GetUserByEmail(email string) (*User, error) {
	var user User
	err := u.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (u *UserCrudService) GetUserByUsername(username string) (*User, error) {
	var user User
	err := u.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	return &user, nil
}
