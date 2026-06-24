package services

import (
	"errors"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const FinalLocalAdminRecoveryMessage = "At least one admin with usable local credentials is required for recovery"

var ErrFinalLocalAdmin = errors.New("final local admin recovery account cannot be removed")

type AdminRecoveryService struct {
	repo     *repository.AdminRepository
	security *SecurityService
}

func NewAdminRecoveryService(repo *repository.AdminRepository, security *SecurityService) *AdminRecoveryService {
	return &AdminRecoveryService{repo: repo, security: security}
}

func (s *AdminRecoveryService) CountLocalRecoveryAdmins() (int64, error) {
	return s.repo.CountLocalRecoveryAdmins()
}

func (s *AdminRecoveryService) EnsureCanDeleteUser(userID uint, actorID *uint) error {
	return s.ensureCanRemoveRecoveryPath(userID, actorID, "delete user")
}

func (s *AdminRecoveryService) EnsureCanDemoteUser(userID uint, actorID *uint) error {
	return s.ensureCanRemoveRecoveryPath(userID, actorID, "demote admin")
}

func (s *AdminRecoveryService) EnsureCanDisableLocalAuth(userID uint, actorID *uint) error {
	return s.ensureCanRemoveRecoveryPath(userID, actorID, "disable local auth")
}

func (s *AdminRecoveryService) EnsureCanClearPassword(userID uint, actorID *uint) error {
	return s.ensureCanRemoveRecoveryPath(userID, actorID, "clear password")
}

func (s *AdminRecoveryService) EnsureCanConvertToOIDCOnly(userID uint, actorID *uint) error {
	return s.ensureCanRemoveRecoveryPath(userID, actorID, "convert to OIDC-only")
}

func (s *AdminRecoveryService) DeleteUserCascade(userID uint, actorID *uint) (int64, error) {
	rows, blockedUser, err := s.repo.DeleteUserCascadeWithRecoveryGuard(userID)
	if errors.Is(err, repository.ErrFinalLocalAdminBlocked) {
		s.recordBlocked(blockedUser, actorID, "delete user")
		return 0, ErrFinalLocalAdmin
	}
	return rows, err
}

func (s *AdminRecoveryService) UpdateUserRole(userID uint, role models.UserRole, actorID *uint) (int64, error) {
	rows, blockedUser, err := s.repo.UpdateUserRoleWithRecoveryGuard(userID, role)
	if errors.Is(err, repository.ErrFinalLocalAdminBlocked) {
		s.recordBlocked(blockedUser, actorID, "demote admin")
		return 0, ErrFinalLocalAdmin
	}
	return rows, err
}

func (s *AdminRecoveryService) ensureCanRemoveRecoveryPath(userID uint, actorID *uint, operation string) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return err
	}
	if !hasUsableLocalAdminCredentials(*user) {
		return nil
	}
	count, err := s.repo.CountLocalRecoveryAdmins()
	if err != nil {
		return err
	}
	if count <= 1 {
		s.recordBlocked(user, actorID, operation)
		return ErrFinalLocalAdmin
	}
	return nil
}

func (s *AdminRecoveryService) recordBlocked(user *models.User, actorID *uint, operation string) {
	if s.security == nil || user == nil {
		return
	}
	userID := user.ID
	adminID := actorID
	if adminID == nil {
		adminID = &userID
	}
	s.security.RecordFinalLocalAdminBlocked(&userID, user.Username, *adminID, operation)
}

func hasUsableLocalAdminCredentials(user models.User) bool {
	return user.Role == models.RoleAdmin && strings.TrimSpace(user.PasswordHash) != ""
}
