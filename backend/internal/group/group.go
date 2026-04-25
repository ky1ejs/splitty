package group

import (
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrNotMember     = errors.New("not a group member")
	ErrAlreadyMember = errors.New("user is already a member")
)

// GroupRecord is the domain model for a group returned from the store.
type GroupRecord struct {
	ID        string
	Name      string
	CreatedBy string
	CreatedAt time.Time
}

// TransactionRecord is the domain model for a transaction returned from the store.
type TransactionRecord struct {
	ID          string
	GroupID     string
	Description string
	Amount      int64
	PaidBy      string
	CreatedAt   time.Time
}
