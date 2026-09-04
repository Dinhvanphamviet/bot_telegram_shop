package service

import (
	"context"
	"log"

	"telegram-shop/internal/model"
	"telegram-shop/internal/repository"

	"github.com/google/uuid"
)

// UserService handles user business logic.
type UserService struct {
	userRepo *repository.UserRepo
}

// NewUserService creates a new UserService.
func NewUserService(userRepo *repository.UserRepo) *UserService {
	return &UserService{userRepo: userRepo}
}

// GetOrCreateUser finds or creates a user from Telegram data.
func (s *UserService) GetOrCreateUser(ctx context.Context, telegramID int64, username, firstName string) (*model.User, error) {
	user, err := s.userRepo.FindByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}

	// Create new user
	user = &model.User{
		TelegramID: telegramID,
		Username:   username,
		FirstName:  firstName,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	log.Printf("Created new user: telegram_id=%d, username=%s", telegramID, username)
	return user, nil
}

// GetUserByTelegramID finds a user by Telegram ID.
func (s *UserService) GetUserByTelegramID(ctx context.Context, telegramID int64) (*model.User, error) {
	return s.userRepo.FindByTelegramID(ctx, telegramID)
}

// GetUserByID finds a user by UUID.
func (s *UserService) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

// GetAllTelegramIDs returns all registered Telegram IDs.
func (s *UserService) GetAllTelegramIDs(ctx context.Context) ([]int64, error) {
	return s.userRepo.FindAllTelegramIDs(ctx)
}

