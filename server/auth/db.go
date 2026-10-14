package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/app"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/user"
	authUtil "git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/authutils"
	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/cerror"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/dig"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ILoginService interface {
	Login(identifier, password, userAgent, ip, requestedSessionId string) (*TokenDto, error)
	Register(username, email, password, userAgent, ip string) (*TokenDto, error)
	RefreshTokens(refreshToken string) (*TokenDto, error)
	LogoutCurrent(userUuid uuid.UUID, sessionUuid uuid.UUID) error
	LogoutAll(userUuid uuid.UUID) error
}

type ISessionService interface {
	ListSessions(userUuid, currentSessionUuid uuid.UUID) ([]SessionDto, error)
	RevokeSession(userUuid, targetSessionUuid uuid.UUID) error
}

type IAuthService interface {
	ILoginService
	ISessionService
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
	svc := &AuthService{
		db:      params.GormDb,
		mongoDb: params.MongoDb,
		logger:  params.Logger,
	}
	svc.RegisterSessionValidator()
	return svc
}

func (s *AuthService) RegisterSessionValidator() {
	if s.mongoDb != nil {
		authUtil.SetSessionValidator(func(ctx context.Context, userUuid, sessionUuid uuid.UUID) bool {
			queryCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			var session user.Session
			err := s.sessionsColl().FindOne(queryCtx, bson.M{
				"user_uuid":    userUuid,
				"session_uuid": sessionUuid,
				"expires_at":   bson.M{"$gt": time.Now()},
			}).Decode(&session)
			return err == nil
		})
	}
}

const defaultContextTimeout = 5 * time.Second

func (s *AuthService) newContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultContextTimeout)
}

func (s *AuthService) sessionsColl() *mongo.Collection {
	return s.mongoDb.Collection("sessions")
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *AuthService) Login(identifier, password, userAgent, ip, requestedSessionId string) (*TokenDto, error) {
	normIdentifier := strings.ToLower(strings.TrimSpace(identifier))
	var usr user.User
	err := s.db.Where("username = ? OR email = ?", normIdentifier, normIdentifier).First(&usr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Debugf("User not found Identifier = %s", normIdentifier)
			return nil, cerror.ErrInvalidCredentials
		}
		s.logger.Errorf("Failed to query user, error = %+v", err)
		return nil, err
	}

	if !authUtil.VerifyPassword(usr.PasswordHash, password) {
		s.logger.Debugf("Invalid password for user Email: %s, uuid: %s", usr.Email, usr.Uuid)
		return nil, cerror.ErrInvalidCredentials
	}

	var sessionUuid uuid.UUID
	var isExistingSession bool

	if requestedSessionId != "" {
		if parsed, err := uuid.Parse(requestedSessionId); err == nil && parsed != uuid.Nil {
			if s.mongoDb != nil {
				ctx, cancel := s.newContext()
				var existingSession user.Session
				err := s.sessionsColl().FindOne(ctx, bson.M{"session_uuid": parsed}).Decode(&existingSession)
				cancel()
				if err == nil {
					if existingSession.UserUuid == usr.Uuid {
						sessionUuid = parsed
						isExistingSession = true
					} else {
						sessionUuid = uuid.New()
					}
				}
			}
		}
	}

	if sessionUuid == uuid.Nil {
		sessionUuid = uuid.New()
	}

	token, refresh, err := authUtil.GenerateTokens(usr.Email, usr.Username, usr.Role, usr.Uuid, sessionUuid)
	if err != nil {
		s.logger.Errorf("Failed to generate token error = %+v", err)
		return nil, err
	}

	if s.mongoDb != nil {
		deviceName := authUtil.ParseDeviceName(userAgent)
		now := time.Now()

		var claims authUtil.Claims
		_, _ = jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
			return []byte(app.AccessKey), nil
		})

		ctx, cancel := s.newContext()
		defer cancel()

		if isExistingSession {
			_, err = s.sessionsColl().UpdateOne(ctx,
				bson.M{"session_uuid": sessionUuid},
				bson.M{
					"$set": bson.M{
						"token_uuid":    claims.TokenUuid,
						"refresh_token": hashToken(refresh),
						"device_name":   deviceName,
						"ip_address":    ip,
						"expires_at":    now.Add(7 * 24 * time.Hour),
					},
				},
			)
			if err != nil {
				s.logger.Errorf("Failed to update existing session, err = %+v", err)
				return nil, err
			}
		} else {
			session := user.Session{
				SessionUuid:  sessionUuid,
				UserUuid:     usr.Uuid,
				TokenUuid:    claims.TokenUuid,
				RefreshToken: hashToken(refresh),
				DeviceName:   deviceName,
				IpAddress:    ip,
				CreatedAt:    now,
				ExpiresAt:    now.Add(7 * 24 * time.Hour),
			}

			_, err = s.sessionsColl().InsertOne(ctx, session)
			if err != nil {
				s.logger.Errorf("Failed to create a session, err = %+v", err)
				return nil, err
			}
		}
	}

	return &TokenDto{
		AccessToken:  token,
		RefreshToken: refresh,
		SessionId:    sessionUuid.String(),
	}, nil
}

func (s *AuthService) Register(username, email, password, userAgent, ip string) (*TokenDto, error) {
	normUsername := strings.ToLower(strings.TrimSpace(username))
	normEmail := strings.ToLower(strings.TrimSpace(email))

	var count int64
	if err := s.db.Model(&user.User{}).Where("username = ? OR email = ?", normUsername, normEmail).Count(&count).Error; err != nil {
		s.logger.Errorf("Failed to check existing user, err = %+v", err)
		return nil, err
	}
	if count > 0 {
		return nil, cerror.ErrUserAlreadyExists
	}

	hash, err := authUtil.HashPassword(password)
	if err != nil {
		s.logger.Errorf("Failed to hash password, err = %+v", err)
		return nil, err
	}

	usr := user.User{
		Uuid:         uuid.New(),
		Username:     normUsername,
		Email:        normEmail,
		PasswordHash: hash,
		Role:         authUtil.ROLE_USER,
	}

	if err := s.db.Create(&usr).Error; err != nil {
		s.logger.Errorf("Failed to create user in db, err = %+v", err)
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, cerror.ErrUserAlreadyExists
		}
		return nil, errors.New("registration failed")
	}

	sessionUuid := uuid.New()
	token, refresh, err := authUtil.GenerateTokens(usr.Email, usr.Username, usr.Role, usr.Uuid, sessionUuid)
	if err != nil {
		s.logger.Errorf("Failed to generate token, err = %+v", err)
		return nil, err
	}

	if s.mongoDb != nil {
		deviceName := authUtil.ParseDeviceName(userAgent)
		now := time.Now()

		var claims authUtil.Claims
		_, _ = jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
			return []byte(app.AccessKey), nil
		})

		session := user.Session{
			SessionUuid:  sessionUuid,
			UserUuid:     usr.Uuid,
			TokenUuid:    claims.TokenUuid,
			RefreshToken: hashToken(refresh),
			DeviceName:   deviceName,
			IpAddress:    ip,
			CreatedAt:    now,
			ExpiresAt:    now.Add(7 * 24 * time.Hour),
		}

		ctx, cancel := s.newContext()
		defer cancel()

		_, err = s.sessionsColl().InsertOne(ctx, session)
		if err != nil {
			s.logger.Errorf("Failed to create a session, err = %+v", err)
			return nil, err
		}
	}

	return &TokenDto{
		AccessToken:  token,
		RefreshToken: refresh,
		SessionId:    sessionUuid.String(),
	}, nil
}

func (s *AuthService) RefreshTokens(rawRefreshToken string) (*TokenDto, error) {
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
		return nil, cerror.ErrInvalidTokenFormat
	}

	parsedUserUuid, err := uuid.Parse(refreshClaims.ID)
	if err != nil {
		s.logger.Errorf("Error Parsing uuid err = %+v", err)
		return nil, err
	}

	if s.mongoDb == nil {
		s.logger.Error("mongoDb is nil in RefreshTokens")
		return nil, errors.New("database connection unavailable")
	}

	ctx, cancel := s.newContext()
	defer cancel()

	hashedInput := hashToken(cleanRefreshToken)
	var session user.Session

	err = s.sessionsColl().FindOne(ctx, bson.M{
		"user_uuid":    parsedUserUuid,
		"session_uuid": refreshClaims.SessionUuid,
	}).Decode(&session)
	if err != nil {
		s.logger.Errorf("Failed to find session by session_uuid: %+v", err)
		return nil, cerror.ErrInvalidTokenFormat
	}

	// Token reuse / replay attack detection
	if session.TokenUuid != refreshClaims.TokenUuid || session.RefreshToken != hashedInput {
		s.logger.Warnf("Token reuse detected for user %s, session %s! Revoking session.", parsedUserUuid, refreshClaims.SessionUuid)
		_, _ = s.sessionsColl().DeleteOne(ctx, bson.M{"session_uuid": refreshClaims.SessionUuid})
		return nil, cerror.ErrInvalidTokenFormat
	}

	var usr user.User
	err = s.db.Where("uuid = ?", parsedUserUuid).First(&usr).Error
	if err != nil {
		s.logger.Errorf("Failed to find user for session: %+v", err)
		return nil, err
	}

	// Generate new tokens preserving the persistent SessionUuid
	newAccessToken, newRefreshToken, err := authUtil.GenerateTokens(usr.Email, usr.Username, usr.Role, usr.Uuid, refreshClaims.SessionUuid)
	if err != nil {
		s.logger.Errorf("Failed to generate tokens, err = %+v", err)
		return nil, err
	}

	var newClaims authUtil.Claims
	_, _ = jwt.ParseWithClaims(newAccessToken, &newClaims, func(t *jwt.Token) (any, error) {
		return []byte(app.AccessKey), nil
	})

	_, err = s.sessionsColl().UpdateOne(ctx,
		bson.M{"session_uuid": refreshClaims.SessionUuid, "token_uuid": refreshClaims.TokenUuid},
		bson.M{
			"$set": bson.M{
				"token_uuid":    newClaims.TokenUuid,
				"refresh_token": hashToken(newRefreshToken),
				"expires_at":    time.Now().Add(7 * 24 * time.Hour),
			},
		},
	)
	if err != nil {
		s.logger.Errorf("Failed to update session: %+v", err)
		return nil, err
	}

	return &TokenDto{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		SessionId:    refreshClaims.SessionUuid.String(),
	}, nil
}

func (s *AuthService) LogoutCurrent(userUuid uuid.UUID, sessionUuid uuid.UUID) error {
	s.logger.Debugf("logging out current session userUuid = %s, sessionUuid = %s", userUuid, sessionUuid)

	if s.mongoDb == nil {
		return nil
	}

	ctx, cancel := s.newContext()
	defer cancel()

	_, err := s.sessionsColl().DeleteOne(ctx, bson.M{
		"user_uuid":    userUuid,
		"session_uuid": sessionUuid,
	})
	if err != nil {
		s.logger.Errorf("Error deleting current session: %+v", err)
		return err
	}

	return nil
}

func (s *AuthService) LogoutAll(userUuid uuid.UUID) error {
	s.logger.Debugf("logging out all sessions for userUuid = %s", userUuid)

	if s.mongoDb == nil {
		return nil
	}

	ctx, cancel := s.newContext()
	defer cancel()

	_, err := s.sessionsColl().DeleteMany(ctx, bson.M{"user_uuid": userUuid})
	if err != nil {
		s.logger.Errorf("Error deleting all sessions: %+v", err)
		return err
	}

	return nil
}

func (s *AuthService) ListSessions(userUuid, currentSessionUuid uuid.UUID) ([]SessionDto, error) {
	if s.mongoDb == nil {
		return []SessionDto{}, nil
	}

	ctx, cancel := s.newContext()
	defer cancel()

	cursor, err := s.sessionsColl().Find(ctx, bson.M{
		"user_uuid":  userUuid,
		"expires_at": bson.M{"$gt": time.Now()},
	})
	if err != nil {
		s.logger.Errorf("Error finding sessions: %+v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var sessions []user.Session
	if err := cursor.All(ctx, &sessions); err != nil {
		s.logger.Errorf("Error decoding sessions: %+v", err)
		return nil, err
	}

	dtos := make([]SessionDto, 0, len(sessions))
	for _, sess := range sessions {
		dtos = append(dtos, SessionDto{
			Id:         sess.SessionUuid.String(),
			DeviceName: sess.DeviceName,
			CreatedAt:  sess.CreatedAt,
			ExpiresAt:  sess.ExpiresAt,
			IsCurrent:  sess.SessionUuid == currentSessionUuid,
		})
	}

	return dtos, nil
}

func (s *AuthService) RevokeSession(userUuid, targetSessionUuid uuid.UUID) error {
	if s.mongoDb == nil {
		return nil
	}

	ctx, cancel := s.newContext()
	defer cancel()

	_, err := s.sessionsColl().DeleteOne(ctx, bson.M{
		"user_uuid":    userUuid,
		"session_uuid": targetSessionUuid,
	})
	if err != nil {
		s.logger.Errorf("Error revoking session: %+v", err)
		return err
	}

	return nil
}
