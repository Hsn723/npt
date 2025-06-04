package pkg

import (
	"context"

	"github.com/cilium/cilium/pkg/identity"
	"github.com/cilium/cilium/pkg/identity/identitymanager"
	"github.com/cilium/cilium/pkg/policy"
	"github.com/cilium/cilium/pkg/policy/api"
)

func InitializeRepo() *policy.Repository {
	return policy.NewPolicyRepository(
		identity.IdentityMap{},
		mockCertificateManager{},
		mockSecretManager{},
		identitymanager.NewIDManager(),
		mockPolicyMetrics{},
	)
}

type mockCertificateManager struct {}

func (m mockCertificateManager) GetTLSContext(_ context.Context, _ *api.TLSContext, _ string) (string, string, string, bool, error) {
	return "", "", "", false, nil
}

type mockSecretManager struct {}

func (m mockSecretManager) GetSecretString(_ context.Context, _ *api.Secret, _ string) (string, error) {
	return "", nil
}

func (m mockSecretManager) PolicySecretSyncEnabled() bool {
	return false
}

func (m mockSecretManager) SecretsOnlyFromSecretsNamespace() bool {
	return false
}

func (m mockSecretManager) GetSecretSyncNamespace() string {
	return ""
}

type mockPolicyMetrics struct {}

func (m mockPolicyMetrics) AddRule(_ api.Rule) {}
func (m mockPolicyMetrics) DelRule(_ api.Rule) {}
