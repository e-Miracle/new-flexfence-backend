package mysql

import (
	"errors"
	"fmt"
	"time"

	"github.com/flexfence/flexfence-backend/internal/auth"
	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
	"gorm.io/gorm"
)

func (s *IdentityStore) CreateUserOTPChallenge(userID, code string, ttl time.Duration) (challengeID string, expiresAt time.Time, err error) {
	hash, err := auth.HashPassword(code)
	if err != nil {
		return "", time.Time{}, err
	}
	now := time.Now().UTC()
	expiresAt = now.Add(ttl)
	challengeID = fmt.Sprintf("uotp_%d", now.UnixNano())

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&UserOTPChallengeModel{}).
			Where("user_id = ? AND consumed_at IS NULL", userID).
			Update("consumed_at", now).Error; err != nil {
			return err
		}
		row := UserOTPChallengeModel{
			ID:        challengeID,
			UserID:    userID,
			CodeHash:  hash,
			ExpiresAt: expiresAt,
			Attempts:  0,
			CreatedAt: now,
		}
		return tx.Create(&row).Error
	})
	return challengeID, expiresAt, err
}

func (s *IdentityStore) ResendUserOTPChallenge(challengeID, code string, ttl time.Duration) (expiresAt time.Time, userID string, email string, err error) {
	var challenge UserOTPChallengeModel
	if err := s.db.Where("id = ?", challengeID).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return time.Time{}, "", "", store.ErrOTPChallengeNotFound
		}
		return time.Time{}, "", "", err
	}
	if challenge.ConsumedAt != nil {
		return time.Time{}, "", "", store.ErrOTPConsumed
	}

	user, ok, err := s.GetUserByID(challenge.UserID)
	if err != nil {
		return time.Time{}, "", "", err
	}
	if !ok {
		return time.Time{}, "", "", store.ErrOTPChallengeNotFound
	}

	hash, err := auth.HashPassword(code)
	if err != nil {
		return time.Time{}, "", "", err
	}
	now := time.Now().UTC()
	expiresAt = now.Add(ttl)
	updates := map[string]any{
		"code_hash":  hash,
		"expires_at": expiresAt,
		"attempts":   0,
	}
	if err := s.db.Model(&UserOTPChallengeModel{}).Where("id = ?", challengeID).Updates(updates).Error; err != nil {
		return time.Time{}, "", "", err
	}
	return expiresAt, user.ID, user.Email, nil
}

func (s *IdentityStore) VerifyUserOTPChallenge(challengeID, code string) (domain.User, error) {
	var challenge UserOTPChallengeModel
	if err := s.db.Where("id = ?", challengeID).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, store.ErrOTPChallengeNotFound
		}
		return domain.User{}, err
	}
	if challenge.ConsumedAt != nil {
		return domain.User{}, store.ErrOTPConsumed
	}
	now := time.Now().UTC()
	if now.After(challenge.ExpiresAt) {
		return domain.User{}, store.ErrOTPExpired
	}
	if challenge.Attempts >= maxOTPAttempts {
		return domain.User{}, store.ErrOTPTooManyAttempts
	}

	if !auth.CheckPassword(challenge.CodeHash, code) {
		_ = s.db.Model(&UserOTPChallengeModel{}).Where("id = ?", challengeID).
			UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error
		return domain.User{}, store.ErrOTPInvalid
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&UserOTPChallengeModel{}).Where("id = ?", challengeID).
			Updates(map[string]any{"consumed_at": now, "attempts": challenge.Attempts + 1}).Error; err != nil {
			return err
		}
		return tx.Model(&UserModel{}).Where("id = ?", challenge.UserID).
			Updates(map[string]any{
				"email_verified": true,
				"updated_at":     now,
			}).Error
	})
	if err != nil {
		return domain.User{}, err
	}

	user, ok, err := s.GetUserByID(challenge.UserID)
	if err != nil {
		return domain.User{}, err
	}
	if !ok {
		return domain.User{}, store.ErrOTPChallengeNotFound
	}
	return user, nil
}

func (s *IdentityStore) GetUserOTPChallengeEmail(challengeID string) (string, error) {
	var challenge UserOTPChallengeModel
	if err := s.db.Where("id = ?", challengeID).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", store.ErrOTPChallengeNotFound
		}
		return "", err
	}
	user, ok, err := s.GetUserByID(challenge.UserID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", store.ErrOTPChallengeNotFound
	}
	return user.Email, nil
}
