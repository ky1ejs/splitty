package model

// Group is a hand-written model so that gqlgen uses it instead of generating one.
// CreatedByID is not exposed in the GraphQL schema — it is used by the CreatedBy field resolver.
type Group struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CreatedByID string `json:"-"`
	CreatedAt   string `json:"createdAt"`
}

// Transaction is a hand-written model so that gqlgen uses it instead of generating one.
// GroupID and PaidByID are not exposed in the GraphQL schema — they are used by field resolvers.
type Transaction struct {
	ID          string `json:"id"`
	GroupID     string `json:"-"`
	Description string `json:"description"`
	Amount      int    `json:"amount"`
	PaidByID    string `json:"-"`
	CreatedAt   string `json:"createdAt"`
}
