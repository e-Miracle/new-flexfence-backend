package mysql

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
	"gorm.io/gorm"
)

var orgSlugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

// IdentityStore handles SaaS tenant and user persistence.
type IdentityStore struct {
	db *gorm.DB
}

func NewIdentityStore(db *gorm.DB) *IdentityStore {
	return &IdentityStore{db: db}
}

func (s *IdentityStore) RegisterBusinessOwner(
	organizationName, email, passwordHash, firstName, lastName string,
) (domain.BusinessUser, error) {
	var created domain.BusinessUser
	err := s.db.Transaction(func(tx *gorm.DB) error {
		email = strings.TrimSpace(strings.ToLower(email))
		var existing BusinessUserModel
		if err := tx.Where("email = ?", email).First(&existing).Error; err == nil {
			return store.ErrAlreadyExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		org, err := createOrganizationTx(tx, organizationName)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		user := BusinessUserModel{
			ID:             fmt.Sprintf("biz_%d", now.UnixNano()),
			OrganizationID: org.ID,
			Email:          email,
			PasswordHash:   passwordHash,
			FirstName:      strings.TrimSpace(firstName),
			LastName:       strings.TrimSpace(lastName),
			Role:           domain.BusinessRoleOwner,
			Status:         domain.StatusActive,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		created = mapBusinessUserModel(user)
		return nil
	})
	if err != nil {
		return domain.BusinessUser{}, err
	}
	return created, nil
}

func createOrganizationTx(tx *gorm.DB, organizationName string) (OrganizationModel, error) {
	base := sanitizeOrgSlug(organizationName)
	if base == "" {
		base = "organization"
	}
	now := time.Now().UTC()
	for i := 0; i < 50; i++ {
		slug := base
		if i > 0 {
			slug = fmt.Sprintf("%s-%d", base, i+1)
		}
		org := OrganizationModel{
			ID:        fmt.Sprintf("org_%d", time.Now().UTC().UnixNano()),
			Name:      strings.TrimSpace(organizationName),
			Slug:      slug,
			Plan:      "trial",
			Status:    domain.StatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tx.Create(&org).Error; err == nil {
			return org, nil
		}
	}
	return OrganizationModel{}, store.ErrInvalidStorageState
}

func sanitizeOrgSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = orgSlugSanitizer.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 48 {
		slug = slug[:48]
	}
	return slug
}

func (s *IdentityStore) CreateOrganization(name, slug, plan string) (domain.Organization, error) {
	now := time.Now().UTC()
	org := OrganizationModel{
		ID:        fmt.Sprintf("org_%d", now.UnixNano()),
		Name:      name,
		Slug:      slug,
		Plan:      plan,
		Status:    domain.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if org.Plan == "" {
		org.Plan = "trial"
	}
	if err := s.db.Create(&org).Error; err != nil {
		return domain.Organization{}, err
	}
	return mapOrganizationModel(org), nil
}

func (s *IdentityStore) CreateBusinessUser(organizationID, email, passwordHash, firstName, lastName, role string) (domain.BusinessUser, error) {
	now := time.Now().UTC()
	if role == "" {
		role = domain.BusinessRoleAdmin
	}
	user := BusinessUserModel{
		ID:             fmt.Sprintf("biz_%d", now.UnixNano()),
		OrganizationID: organizationID,
		Email:          email,
		PasswordHash:   passwordHash,
		FirstName:      firstName,
		LastName:       lastName,
		Role:           role,
		Status:         domain.StatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return domain.BusinessUser{}, err
	}
	return mapBusinessUserModel(user), nil
}

func (s *IdentityStore) GetBusinessUserByID(id string) (domain.BusinessUser, bool, error) {
	var user BusinessUserModel
	err := s.db.Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.BusinessUser{}, false, nil
	}
	if err != nil {
		return domain.BusinessUser{}, false, err
	}
	return mapBusinessUserModel(user), true, nil
}

func (s *IdentityStore) GetUserByID(id string) (domain.User, bool, error) {
	var user UserModel
	err := s.db.Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, false, nil
	}
	if err != nil {
		return domain.User{}, false, err
	}
	return mapUserModel(user), true, nil
}

func (s *IdentityStore) GetBusinessUserByEmail(email string) (domain.BusinessUser, bool, error) {
	var user BusinessUserModel
	err := s.db.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.BusinessUser{}, false, nil
	}
	if err != nil {
		return domain.BusinessUser{}, false, err
	}
	return mapBusinessUserModel(user), true, nil
}

type BusinessAuthRecord struct {
	User         domain.BusinessUser
	PasswordHash string
}

func (s *IdentityStore) GetBusinessAuthByEmail(email string) (BusinessAuthRecord, bool, error) {
	var user BusinessUserModel
	err := s.db.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return BusinessAuthRecord{}, false, nil
	}
	if err != nil {
		return BusinessAuthRecord{}, false, err
	}
	return BusinessAuthRecord{
		User:         mapBusinessUserModel(user),
		PasswordHash: user.PasswordHash,
	}, true, nil
}

func (s *IdentityStore) TouchBusinessLogin(userID string) error {
	now := time.Now().UTC()
	return s.db.Model(&BusinessUserModel{}).Where("id = ?", userID).Updates(map[string]any{
		"last_login_at": now,
		"updated_at":    now,
	}).Error
}

func (s *IdentityStore) GetUserByGoogleSub(googleSub string) (domain.User, bool, error) {
	var user UserModel
	err := s.db.Where("google_sub = ?", googleSub).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, false, nil
	}
	if err != nil {
		return domain.User{}, false, err
	}
	return mapUserModel(user), true, nil
}

func (s *IdentityStore) UpsertUserFromGoogle(googleSub, email, firstName, lastName string) (domain.User, error) {
	if existing, ok, err := s.GetUserByGoogleSub(googleSub); err != nil {
		return domain.User{}, err
	} else if ok {
		return existing, nil
	}

	if byEmail, ok, err := s.GetUserByEmail(email); err != nil {
		return domain.User{}, err
	} else if ok {
		now := time.Now().UTC()
		if err := s.db.Model(&UserModel{}).Where("id = ?", byEmail.ID).Updates(map[string]any{
			"google_sub":     googleSub,
			"first_name":     firstName,
			"last_name":      lastName,
			"email_verified": true,
			"updated_at":     now,
		}).Error; err != nil {
			return domain.User{}, err
		}
		byEmail, _, _ = s.GetUserByEmail(email)
		return byEmail, nil
	}

	return s.CreateUserWithGoogle(googleSub, email, firstName, lastName)
}

func (s *IdentityStore) CreateUserWithGoogle(googleSub, email, firstName, lastName string) (domain.User, error) {
	now := time.Now().UTC()
	user := UserModel{
		ID:            fmt.Sprintf("usr_%d", now.UnixNano()),
		Email:         email,
		FirstName:     firstName,
		LastName:      lastName,
		GoogleSub:     googleSub,
		EmailVerified: true,
		Status:        domain.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return domain.User{}, err
	}
	return mapUserModel(user), nil
}

func (s *IdentityStore) CreateUser(email, firstName, lastName, phone string) (domain.User, error) {
	now := time.Now().UTC()
	user := UserModel{
		ID:        fmt.Sprintf("usr_%d", now.UnixNano()),
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Phone:     phone,
		Status:    domain.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return domain.User{}, err
	}
	return mapUserModel(user), nil
}

type UserAuthRecord struct {
	User         domain.User
	PasswordHash string
}

func (s *IdentityStore) GetUserAuthByEmail(email string) (UserAuthRecord, bool, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	var user UserModel
	err := s.db.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return UserAuthRecord{}, false, nil
	}
	if err != nil {
		return UserAuthRecord{}, false, err
	}
	return UserAuthRecord{
		User:         mapUserModel(user),
		PasswordHash: user.PasswordHash,
	}, true, nil
}

func (s *IdentityStore) GetUserAuthByID(userID string) (UserAuthRecord, bool, error) {
	var user UserModel
	err := s.db.Where("id = ?", userID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return UserAuthRecord{}, false, nil
	}
	if err != nil {
		return UserAuthRecord{}, false, err
	}
	return UserAuthRecord{
		User:         mapUserModel(user),
		PasswordHash: user.PasswordHash,
	}, true, nil
}

func (s *IdentityStore) UpdateUserProfile(userID, firstName, lastName string) (domain.User, error) {
	now := time.Now().UTC()
	res := s.db.Model(&UserModel{}).Where("id = ?", userID).Updates(map[string]any{
		"first_name": strings.TrimSpace(firstName),
		"last_name":  strings.TrimSpace(lastName),
		"updated_at": now,
	})
	if res.Error != nil {
		return domain.User{}, res.Error
	}
	if res.RowsAffected == 0 {
		return domain.User{}, store.ErrUserNotFound
	}
	user, ok, err := s.GetUserByID(userID)
	if err != nil {
		return domain.User{}, err
	}
	if !ok {
		return domain.User{}, store.ErrUserNotFound
	}
	return user, nil
}

func (s *IdentityStore) UpdateUserPasswordHash(userID, passwordHash string) error {
	now := time.Now().UTC()
	res := s.db.Model(&UserModel{}).Where("id = ?", userID).Updates(map[string]any{
		"password_hash": passwordHash,
		"updated_at":      now,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return store.ErrUserNotFound
	}
	return nil
}

func (s *IdentityStore) RegisterUserWithPassword(email, passwordHash, firstName, lastName, phone string) (domain.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	var existing UserModel
	if err := s.db.Where("email = ?", email).First(&existing).Error; err == nil {
		return domain.User{}, store.ErrAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, err
	}
	now := time.Now().UTC()
	user := UserModel{
		ID:           fmt.Sprintf("usr_%d", now.UnixNano()),
		Email:        email,
		PasswordHash: passwordHash,
		FirstName:    strings.TrimSpace(firstName),
		LastName:     strings.TrimSpace(lastName),
		Phone:        strings.TrimSpace(phone),
		Status:       domain.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return domain.User{}, err
	}
	return mapUserModel(user), nil
}

func (s *IdentityStore) GetUserByEmail(email string) (domain.User, bool, error) {
	var user UserModel
	err := s.db.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, false, nil
	}
	if err != nil {
		return domain.User{}, false, err
	}
	return mapUserModel(user), true, nil
}

// DeleteAttendeeUser permanently removes an attendee and related mobile data.
func (s *IdentityStore) DeleteAttendeeUser(userID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&EventJoinModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&AttendanceModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&UserActivitySessionModel{}).Error; err != nil {
			return err
		}
		res := tx.Where("id = ?", userID).Delete(&UserModel{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return store.ErrUserNotFound
		}
		return nil
	})
}

func (s *IdentityStore) LoginBusinessWithGoogle(googleSub, email, firstName, lastName string) (domain.BusinessUser, error) {
	record, ok, err := s.GetBusinessAuthByEmail(email)
	if err != nil {
		return domain.BusinessUser{}, err
	}
	if !ok {
		return domain.BusinessUser{}, store.ErrInvalidCredentials
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"google_sub": googleSub,
		"updated_at": now,
	}
	if firstName != "" {
		updates["first_name"] = firstName
	}
	if lastName != "" {
		updates["last_name"] = lastName
	}
	if err := s.db.Model(&BusinessUserModel{}).Where("id = ?", record.User.ID).Updates(updates).Error; err != nil {
		return domain.BusinessUser{}, err
	}
	user, ok, err := s.GetBusinessUserByID(record.User.ID)
	if err != nil || !ok {
		return domain.BusinessUser{}, err
	}
	return user, nil
}

func mapOrganizationModel(m OrganizationModel) domain.Organization {
	return domain.Organization{
		ID:        m.ID,
		Name:      m.Name,
		Slug:      m.Slug,
		Plan:      m.Plan,
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func mapBusinessUserModel(m BusinessUserModel) domain.BusinessUser {
	return domain.BusinessUser{
		ID:             m.ID,
		OrganizationID: m.OrganizationID,
		Email:          m.Email,
		FirstName:      m.FirstName,
		LastName:       m.LastName,
		Role:           m.Role,
		Status:         m.Status,
		LastLoginAt:    m.LastLoginAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func mapUserModel(m UserModel) domain.User {
	return domain.User{
		ID:            m.ID,
		Email:         m.Email,
		FirstName:     m.FirstName,
		LastName:      m.LastName,
		Phone:         m.Phone,
		EmailVerified: m.EmailVerified,
		Status:        m.Status,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}
