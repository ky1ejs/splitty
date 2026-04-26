package dataloader

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kylejs/splitty/backend/internal/auth"
	"github.com/kylejs/splitty/backend/internal/group"
	"github.com/vikstrous/dataloadgen"
)

type ctxKey struct{}

// Loaders holds all request-scoped dataloaders.
type Loaders struct {
	UserLoader              *dataloadgen.Loader[string, *auth.UserRecord]
	GroupLoader             *dataloadgen.Loader[string, *group.GroupRecord]
	GroupMembersLoader      *dataloadgen.Loader[string, []string]
	TransactionSplitsLoader *dataloadgen.Loader[string, []string]
}

// NewLoaders creates a fresh set of dataloaders backed by the given stores.
func NewLoaders(userStore *auth.PgUserStore, groupStore *group.PgGroupStore) *Loaders {
	return &Loaders{
		UserLoader:              dataloadgen.NewLoader(userFetch(userStore)),
		GroupLoader:             dataloadgen.NewLoader(groupFetch(groupStore)),
		GroupMembersLoader:      dataloadgen.NewMappedLoader(groupMembersFetch(groupStore)),
		TransactionSplitsLoader: dataloadgen.NewMappedLoader(transactionSplitsFetch(groupStore)),
	}
}

// Middleware creates per-request dataloaders and stores them in context.
func Middleware(userStore *auth.PgUserStore, groupStore *group.PgGroupStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			loaders := NewLoaders(userStore, groupStore)
			ctx := context.WithValue(r.Context(), ctxKey{}, loaders)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// For extracts the dataloaders from the request context.
func For(ctx context.Context) *Loaders {
	return ctx.Value(ctxKey{}).(*Loaders)
}

func userFetch(store *auth.PgUserStore) func(context.Context, []string) ([]*auth.UserRecord, []error) {
	return func(ctx context.Context, keys []string) ([]*auth.UserRecord, []error) {
		records, err := store.GetByIDs(ctx, keys)
		if err != nil {
			errs := make([]error, len(keys))
			for i := range errs {
				errs[i] = err
			}
			return nil, errs
		}
		byID := make(map[string]*auth.UserRecord, len(records))
		for _, r := range records {
			byID[r.ID] = r
		}
		results := make([]*auth.UserRecord, len(keys))
		errs := make([]error, len(keys))
		for i, key := range keys {
			if r, ok := byID[key]; ok {
				results[i] = r
			} else {
				errs[i] = fmt.Errorf("user %s not found", key)
			}
		}
		return results, errs
	}
}

func groupFetch(store *group.PgGroupStore) func(context.Context, []string) ([]*group.GroupRecord, []error) {
	return func(ctx context.Context, keys []string) ([]*group.GroupRecord, []error) {
		records, err := store.GetByIDs(ctx, keys)
		if err != nil {
			errs := make([]error, len(keys))
			for i := range errs {
				errs[i] = err
			}
			return nil, errs
		}
		byID := make(map[string]*group.GroupRecord, len(records))
		for _, r := range records {
			byID[r.ID] = r
		}
		results := make([]*group.GroupRecord, len(keys))
		errs := make([]error, len(keys))
		for i, key := range keys {
			if r, ok := byID[key]; ok {
				results[i] = r
			} else {
				errs[i] = group.ErrNotFound
			}
		}
		return results, errs
	}
}

func groupMembersFetch(store *group.PgGroupStore) func(context.Context, []string) (map[string][]string, error) {
	return func(ctx context.Context, keys []string) (map[string][]string, error) {
		return store.GetMembersByGroupIDs(ctx, keys)
	}
}

func transactionSplitsFetch(store *group.PgGroupStore) func(context.Context, []string) (map[string][]string, error) {
	return func(ctx context.Context, keys []string) (map[string][]string, error) {
		return store.GetSplitUserIDsByTransactionIDs(ctx, keys)
	}
}
