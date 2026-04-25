package graph

import (
	"context"
	"fmt"

	"github.com/kylejs/splitty/backend/graph/model"
	"github.com/kylejs/splitty/backend/internal/auth"
	"github.com/kylejs/splitty/backend/internal/group"
)

func userRecordToModel(u *auth.UserRecord) *model.User {
	return &model.User{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
	}
}

// requireAuth extracts the authenticated user ID from the context.
func (r *Resolver) requireAuth(ctx context.Context) (string, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return "", fmt.Errorf("authentication required")
	}
	return userID, nil
}

// requireGroupMember extracts the authenticated user and verifies group membership.
func (r *Resolver) requireGroupMember(ctx context.Context, groupID string) (string, error) {
	userID, err := r.requireAuth(ctx)
	if err != nil {
		return "", err
	}
	isMember, err := r.GroupStore.IsMember(ctx, groupID, userID)
	if err != nil {
		return "", fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return "", fmt.Errorf("not a member of this group")
	}
	return userID, nil
}

func groupRecordToModel(g *group.GroupRecord) *model.Group {
	return &model.Group{
		ID:          g.ID,
		Name:        g.Name,
		CreatedByID: g.CreatedBy,
		CreatedAt:   g.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func txnRecordToModel(t *group.TransactionRecord) *model.Transaction {
	return &model.Transaction{
		ID:          t.ID,
		GroupID:     t.GroupID,
		Description: t.Description,
		Amount:      int(t.Amount),
		PaidByID:    t.PaidBy,
		CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
