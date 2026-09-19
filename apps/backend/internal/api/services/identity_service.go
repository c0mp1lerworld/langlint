package services

import (
	"context"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

// IdentityService serves the A9 user-data endpoints: portability, right to be
// forgotten and the access audit. The single-user identity (id + email) comes
// from configuration in the MVP (no auth, PRODUCT_DOMAIN §3.2).
type IdentityService struct {
	email     identity.Email
	practices storage.PracticeRepository
	deletions storage.DeletionRequestRepository
	accessLog storage.AccessLogRepository
	now       func() time.Time
}

// NewIdentityService wires the A9 use cases through their ports.
func NewIdentityService(
	email identity.Email,
	practices storage.PracticeRepository,
	deletions storage.DeletionRequestRepository,
	accessLog storage.AccessLogRepository,
) *IdentityService {
	return &IdentityService{
		email:     email,
		practices: practices,
		deletions: deletions,
		accessLog: accessLog,
		now:       time.Now,
	}
}

// Email returns the portable identity email carried by the export.
func (s *IdentityService) Email() identity.Email { return s.email }

// Export returns every practice owned by the user (GDPR Art. 15+20, A9).
func (s *IdentityService) Export(ctx context.Context, userID domain.ID) ([]practice.Practice, error) {
	return s.practices.ListAllByUser(ctx, userID)
}

// RequestDeletion appends a pending deletion request. Requesting again while one
// is already pending is an idempotent success (AP4).
func (s *IdentityService) RequestDeletion(ctx context.Context, userID domain.ID) error {
	pending, err := s.deletions.HasPending(ctx, userID)
	if err != nil {
		return err
	}
	if pending {
		return nil
	}

	req, err := identity.NewDeletionRequest(userID, s.now())
	if err != nil {
		return err
	}
	return s.deletions.Append(ctx, req)
}

// AccessLog returns a page of the user's append-only access audit (A9, A4).
func (s *IdentityService) AccessLog(ctx context.Context, userID domain.ID, limit, offset int) ([]identity.AccessEvent, int, error) {
	return s.accessLog.ListByUser(ctx, userID, limit, offset)
}
