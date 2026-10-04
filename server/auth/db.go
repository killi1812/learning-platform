package auth

import (
	"context"
	"errors"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	authUtil "git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/auth"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/dig"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IAuthService interface {
	Login(identifier, password string) (string, string, error)
	RefreshTokens(refreshToken string) (string, string, error)
	Logout(userUuid string) error
}

type AuthService struct {
	db      *gorm.DB
	mongoDb *mongo.Database
	logger  *zap.SugaredLogger
}

type AuthServiceParams struct {
	dig.In
	GormDb  *gorm.DB
	MongoDb *mongo.Database `name:"mongo"`
	Logger  *zap.SugaredLogger
}

func NewAuthService(params AuthServiceParams) IAuthService {
	return &AuthService{
		db:      params.GormDb,
		mongoDb: params.MongoDb,
		logger:  params.Logger,
	}
}

func (s *AuthService) sessionsColl() *mongo.Collection {
	return s.mongoDb.Collection("sessions")
}

func (s *AuthService) Login(identifier, password string) (string, string, error) {
	var usr user.User
	err := s.db.Where("username = ? OR email = ?", identifier, identifier).First(&usr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Debugf("User not found Identifier = %s", identifier)
			return "", "", cerror.ErrInvalidCredentials
		}
		s.logger.Errorf("Failed to query user, error = %+v", err)
		return "", "", err
	}

	if !authUtil.VerifyPassword(usr.PasswordHash, password) {
		s.logger.Debugf("Invalid password for user Email: %s, uuid: %s", usr.Email, usr.Uuid)
		return "", "", cerror.ErrInvalidCredentials
	}

	token, refresh, err := authUtil.GenerateTokens(usr.Email, usr.Username, string(usr.Role), usr.Uuid)
	if err != nil {
		s.logger.Errorf("Failed to generate token error = %+v", err)
		return "", "", err
	}

	// Remove previous session if exists
	_ = s.Logout(usr.Uuid.String())

	session := user.Session{
		UserUuid:     usr.Uuid,
		RefreshToken: refresh,
		CreatedAt:    time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = s.sessionsColl().InsertOne(ctx, session)
	if err != nil {
		s.logger.Errorf("Failed to create a session, err = %+v", err)
		return "", "", err
	}

	return token, refresh, nil
}

func (s *AuthService) RefreshTokens(rawRefreshToken string) (string, string, error) {
	cleanRefreshToken, err := authUtil.ExtractToken(rawRefreshToken)
	if err != nil {
		cleanRefreshToken = rawRefreshToken
	}

	var refreshClaims authUtil.Claims
	token, err := jwt.ParseWithClaims(cleanRefreshToken, &refreshClaims, func(token *jwt.Token) (any, error) {
		return []byte(app.RefreshKey), nil
	})
	if err != nil || !token.Valid {
		s.logger.Errorf("Error parsing refresh token claims, err = %+v", err)
		return "", "", cerror.ErrInvalidTokenFormat
	}

	parsedUserUuid, err := uuid.Parse(refreshClaims.ID)
	if err != nil {
		s.logger.Errorf("Error Parsing uuid err = %+v", err)
		return "", "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var session user.Session
	err = s.sessionsColl().FindOne(ctx, bson.M{
		"user_uuid":     parsedUserUuid,
		"refresh_token": cleanRefreshToken,
	}).Decode(&session)
	if err != nil {
		s.logger.Errorf("Failed to find session, err = %+v", err)
		return "", "", cerror.ErrInvalidTokenFormat
	}

	var usr user.User
	err = s.db.Where("uuid = ?", parsedUserUuid).First(&usr).Error
	if err != nil {
		s.logger.Errorf("Failed to find user for session: %+v", err)
		return "", "", err
	}

	newAccessToken, newRefreshToken, err := authUtil.GenerateTokens(usr.Email, usr.Username, string(usr.Role), usr.Uuid)
	if err != nil {
		s.logger.Errorf("Failed to generate tokens, err = %+v", err)
		return "", "", err
	}

	session.RefreshToken = newRefreshToken
	_, err = s.sessionsColl().ReplaceOne(ctx, bson.M{"user_uuid": parsedUserUuid}, session)
	if err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}

func (s *AuthService) Logout(userUuidStr string) error {
	s.logger.Debugf("logging out user with uuid = %s", userUuidStr)
	parsedUuid, err := uuid.Parse(userUuidStr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = s.sessionsColl().DeleteMany(ctx, bson.M{"user_uuid": parsedUuid})
	if err != nil {
		s.logger.Errorf("Error deleting session: %+v", err)
		return err
	}

	return nil
}
