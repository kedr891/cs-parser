package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/api/repository"
	"github.com/kedr891/cs-parser/internal/domain"
	"github.com/kedr891/cs-parser/internal/entity"
)

type UserService struct {
	repo      repository.UserRepository
	cache     domain.CacheStorage
	jwtSecret string
	log       domain.Logger
}

func NewUserService(
	repo repository.UserRepository,
	cache domain.CacheStorage,
	jwtSecret string,
	log domain.Logger,
) *UserService {
	return &UserService{
		repo:      repo,
		cache:     cache,
		jwtSecret: jwtSecret,
		log:       log,
	}
}

func (s *UserService) Register(ctx context.Context, req *entity.RegisterRequest) (*entity.LoginResponse, error) {
	exists, err := s.repo.UserExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("check user existence: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("user with this email already exists")
	}

	user, err := entity.NewUser(req.Email, req.Username, req.Password)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	settings := entity.NewUserSettings(user.ID)
	if err := s.repo.CreateUserSettings(ctx, settings); err != nil {
		s.log.Warn("Failed to create user settings", "error", err)
	}

	token, err := s.generateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &entity.LoginResponse{
		User:  user.Sanitize(),
		Token: token,
	}, nil
}

func (s *UserService) Login(ctx context.Context, req *entity.LoginRequest) (*entity.LoginResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if !user.CheckPassword(req.Password) {
		return nil, fmt.Errorf("invalid email or password")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	user.UpdateLastLogin()
	if err := s.repo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.log.Warn("Failed to update last login", "error", err)
	}

	token, err := s.generateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &entity.LoginResponse{
		User:  user.Sanitize(),
		Token: token,
	}, nil
}

func (s *UserService) GetProfile(ctx context.Context, userID uuid.UUID) (*entity.UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	return user.Sanitize(), nil
}

func (s *UserService) GetWatchlist(ctx context.Context, userID uuid.UUID) ([]entity.WatchlistWithSkin, error) {
	return s.repo.GetWatchlistWithSkins(ctx, userID)
}

func (s *UserService) AddToWatchlist(ctx context.Context, userID, skinID uuid.UUID, targetPrice *float64, notifyOnDrop bool) (*entity.Watchlist, error) {
	exists, err := s.repo.WatchlistExists(ctx, userID, skinID)
	if err != nil {
		return nil, fmt.Errorf("check watchlist: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("skin already in watchlist")
	}

	watchlist := entity.NewWatchlist(userID, skinID)
	if targetPrice != nil {
		watchlist.SetTargetPrice(*targetPrice)
	}
	watchlist.NotifyOnDrop = notifyOnDrop

	if err := s.repo.CreateWatchlist(ctx, watchlist); err != nil {
		return nil, fmt.Errorf("create watchlist: %w", err)
	}

	return watchlist, nil
}

func (s *UserService) RemoveFromWatchlist(ctx context.Context, userID, skinID uuid.UUID) error {
	return s.repo.DeleteWatchlist(ctx, userID, skinID)
}

func (s *UserService) GetNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit int) (*entity.NotificationListResponse, error) {
	notifications, err := s.repo.GetNotifications(ctx, userID, unreadOnly, limit)
	if err != nil {
		return nil, fmt.Errorf("get notifications: %w", err)
	}

	unreadCount, err := s.repo.GetUnreadCount(ctx, userID)
	if err != nil {
		s.log.Warn("Failed to get unread count", "error", err)
		unreadCount = 0
	}

	return &entity.NotificationListResponse{
		Notifications: notifications,
		Total:         len(notifications),
		Unread:        unreadCount,
		Page:          1,
		PageSize:      limit,
	}, nil
}

func (s *UserService) MarkNotificationsRead(ctx context.Context, userID uuid.UUID, notificationIDs []uuid.UUID) error {
	return s.repo.MarkNotificationsRead(ctx, userID, notificationIDs)
}

func (s *UserService) GetUserStats(ctx context.Context, userID uuid.UUID) (*entity.UserStats, error) {
	return s.repo.GetUserStats(ctx, userID)
}

func (s *UserService) generateJWT(user *entity.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(168 * time.Hour).Unix(), // 7 days
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
