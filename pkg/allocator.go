package pkg

import (
	"context"
	"log/slog"

	"github.com/cilium/cilium/pkg/identity"
	"github.com/cilium/cilium/pkg/identity/cache"
	"github.com/cilium/cilium/pkg/k8s/client/clientset/versioned/fake"
	"github.com/cilium/cilium/pkg/kvstore"
	"github.com/cilium/statedb"
)

type mockIdentityAllocatorOwner struct{}

func (_ mockIdentityAllocatorOwner) UpdateIdentities(added, deleted identity.IdentityMap) <-chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}

func (_ mockIdentityAllocatorOwner) GetNodeSuffix() string {
	return ""
}

type fakeKVLocker struct {}

func (f fakeKVLocker) Unlock(ctx context.Context) error {
	return nil
}
func (f fakeKVLocker) Comparator() any {
	return nil
}

type fakeClient struct {
	kvstore.Client
}

func (f fakeClient) LockPath(ctx context.Context, path string) (kvstore.KVLocker, error) {
	return fakeKVLocker{}, nil
}

func getMockIdentityAllocator(logger *slog.Logger, db *statedb.DB) *cache.CachingIdentityAllocator {
	owner := mockIdentityAllocatorOwner{}
	allocator := cache.NewCachingIdentityAllocator(logger, owner, cache.NewTestAllocatorConfig())
	kvc := fakeClient{kvstore.NewInMemoryClient(db, "npt-cluster")}
	allocator.InitIdentityAllocator(fake.NewSimpleClientset(), kvc)
	return allocator
}
