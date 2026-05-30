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

const maxOTPAttempts = 5

// CreateBusinessOTPChallenge invalidates prior open challenges and stores a new code.
func (s *IdentityStore) CreateBusinessOTPChallenge(businessUserID, code string, ttl time.Duration) (challengeID string, expiresAt time.Time, err error) {
	hash, err := auth.HashPassword(code)
	if err != nil {
		return "", time.Time{}, err
	}
	now := time.Now().UTC()
	expiresAt = now.Add(ttl)
	challengeID = fmt.Sprintf("otp_%d", now.UnixNano())

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&BusinessOTPChallengeModel{}).
			Where("business_user_id = ? AND consumed_at IS NULL", businessUserID).
			Update("consumed_at", now).Error; err != nil {
			return err
		}
		row := BusinessOTPChallengeModel{
			ID:             challengeID,
			BusinessUserID: businessUserID,
			CodeHash:       hash,
			ExpiresAt:      expiresAt,
			Attempts:       0,
			CreatedAt:      now,
		}
		return tx.Create(&row).Error
	})
	return challengeID, expiresAt, err
}

// ResendBusinessOTPChallenge replaces the code on an active challenge.
func (s *IdentityStore) ResendBusinessOTPChallenge(challengeID, code string, ttl time.Duration) (expiresAt time.Time, businessUserID string, email string, err error) {
	var challenge BusinessOTPChallengeModel
	if err := s.db.Where("id = ?", challengeID).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return time.Time{}, "", "", store.ErrOTPChallengeNotFound
		}
		return time.Time{}, "", "", err
	}
	if challenge.ConsumedAt != nil {
		return time.Time{}, "", "", store.ErrOTPConsumed
	}

	user, ok, err := s.GetBusinessUserByID(challenge.BusinessUserID)
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
	if err := s.db.Model(&BusinessOTPChallengeModel{}).Where("id = ?", challengeID).Updates(updates).Error; err != nil {
		return time.Time{}, "", "", err
	}
	return expiresAt, user.ID, user.Email, nil
}

// VerifyBusinessOTPChallenge validates a code and marks the challenge consumed.
func (s *IdentityStore) VerifyBusinessOTPChallenge(challengeID, code string) (domain.BusinessUser, error) {
	var challenge BusinessOTPChallengeModel
	if err := s.db.Where("id = ?", challengeID).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.BusinessUser{}, store.ErrOTPChallengeNotFound
		}
		return domain.BusinessUser{}, err
	}
	if challenge.ConsumedAt != nil {
		return domain.BusinessUser{}, store.ErrOTPConsumed
	}
	now := time.Now().UTC()
	if now.After(challenge.ExpiresAt) {
		return domain.BusinessUser{}, store.ErrOTPExpired
	}
	if challenge.Attempts >= maxOTPAttempts {
		return domain.BusinessUser{}, store.ErrOTPTooManyAttempts
	}

	if !auth.CheckPassword(challenge.CodeHash, code) {
		_ = s.db.Model(&BusinessOTPChallengeModel{}).Where("id = ?", challengeID).
			UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error
		return domain.BusinessUser{}, store.ErrOTPInvalid
	}

	if err := s.db.Model(&BusinessOTPChallengeModel{}).Where("id = ?", challengeID).
		Updates(map[string]any{"consumed_at": now, "attempts": challenge.Attempts + 1}).Error; err != nil {
		return domain.BusinessUser{}, err
	}

	user, ok, err := s.GetBusinessUserByID(challenge.BusinessUserID)
	if err != nil {
		return domain.BusinessUser{}, err
	}
	if !ok {
		return domain.BusinessUser{}, store.ErrOTPChallengeNotFound
	}
	return user, nil
}

// GetBusinessOTPChallengeEmail returns the business user email for an active challenge.
func (s *IdentityStore) GetBusinessOTPChallengeEmail(challengeID string) (string, error) {
	var challenge BusinessOTPChallengeModel
	if err := s.db.Where("id = ?", challengeID).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", store.ErrOTPChallengeNotFound
		}
		return "", err
	}
	user, ok, err := s.GetBusinessUserByID(challenge.BusinessUserID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", store.ErrOTPChallengeNotFound
	}
	return user.Email, nil
}
