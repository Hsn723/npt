package pkg

import (
	"context"
	"log/slog"

	"github.com/cilium/cilium/pkg/crypto/certificatemanager"
	"github.com/cilium/cilium/pkg/identity"
	"github.com/cilium/cilium/pkg/identity/identitymanager"
	envoypolicy "github.com/cilium/cilium/pkg/envoy/policy"
	"github.com/cilium/cilium/pkg/policy"
	"github.com/cilium/cilium/pkg/policy/api"
)

func InitializeRepo(logger *slog.Logger, idm identitymanager.IDManager) *policy.Repository {
	return policy.NewPolicyRepository(
		logger,
		identity.ListReservedIdentities(),
		mockCertificateManager{},
		envoypolicy.NewEnvoyL7RulesTranslator(logger, certificatemanager.NewMockSecretManagerInline()),
		idm,
		mockPolicyMetrics{},
	)
}

type mockCertificateManager struct {}

func (m mockCertificateManager) GetTLSContext(_ context.Context, _ *api.TLSContext, _ string) (string, string, string, bool, error) {
	return "", "", "", false, nil
}

type mockPolicyMetrics struct {}

func (m mockPolicyMetrics) AddRule(_ api.Rule) {}
func (m mockPolicyMetrics) DelRule(_ api.Rule) {}
