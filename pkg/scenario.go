package pkg

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/cilium/cilium/api/v1/models"
	"github.com/cilium/cilium/pkg/identity"
	"github.com/cilium/cilium/pkg/identity/cache"
	"github.com/cilium/cilium/pkg/identity/identitymanager"
	"github.com/cilium/cilium/pkg/labels"
	"github.com/cilium/cilium/pkg/policy"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/cilium/cilium/pkg/u8proto"
	"github.com/cilium/statedb"
	"golang.org/x/sync/errgroup"
)

type Scenario struct {
	Name            string         `json:"name"`
	From            []labels.Label `json:"from"`
	To              []labels.Label `json:"to"`
	DPort           *models.Port   `json:"dPort"`
	Direction       Direction      `json:"direction"`
	ExpectedVerdict Verdict        `json:"expectedVerdict"`
}

func (s Scenario) ToFlow(allocator *cache.CachingIdentityAllocator) *policy.Flow {
	ctx := context.Background()
	protocol, err := u8proto.ParseProtocol(s.DPort.Protocol)
	if err != nil {
		return nil
	}
	fromIdentity, isAlloc, err := allocator.AllocateIdentity(ctx, labels.FromSlice(s.From), false, identity.InvalidIdentity)
	if err != nil || !isAlloc {
		return nil
	}
	toIdentity, isAlloc, err := allocator.AllocateIdentity(ctx, labels.FromSlice(s.To), false, identity.InvalidIdentity)
	if err != nil || !isAlloc {
		return nil
	}
	flow := &policy.Flow{
		From:  fromIdentity,
		To:    toIdentity,
		Proto: protocol,
		Dport: s.DPort.Port,
	}
	return flow
}

func (s Scenario) ToEndpointInfo(logger *slog.Logger, flow *policy.Flow) (src, dest *policy.EndpointInfo) {
	src = &policy.EndpointInfo{
		ID: uint64(flow.From.ID),
		Logger: logger,
	}
	dest = &policy.EndpointInfo{
		ID: uint64(flow.To.ID),
		Logger: logger,
	}
	namedPort := map[string]uint16{
		s.DPort.Name: s.DPort.Port,
	}
	switch s.DPort.Protocol {
	case "TCP":
		src.TCPNamedPorts = namedPort
	case "UDP":
		src.UDPNamedPorts = namedPort
	}
	return src, dest
}

type Direction string

const (
	DirectionIngress Direction = "ingress"
	DirectionEgress  Direction = "egress"
)

type Verdict string

const (
	VerdictAllow     Verdict = "Allowed"
	VerdictDeny      Verdict = "Denied"
	VerdictUndecided Verdict = "Undecided"
)

func (v Verdict) Decision() api.Decision {
	switch v {
	case VerdictAllow:
		return api.Allowed
	case VerdictDeny:
		return api.Denied
	default:
		return api.Undecided
	}
}

func (v Verdict) String() string {
	return string(v)
}

func ParseDecision(d api.Decision) Verdict {
	switch d {
	case api.Allowed:
		return VerdictAllow
	case api.Denied:
		return VerdictDeny
	default:
		return VerdictUndecided
	}
}

type ScenarioResult struct {
	Name     string  `json:"name"`
	Expected Verdict `json:"expectedVerdict"`
	Actual   Verdict `json:"actualVerdict"`
}

func LoadScenarios(scenarioDirs, scenarioFiles []string) ([]Scenario, error) {
	var allScenarios []Scenario
	loader := func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		scenarioFiles = append(scenarioFiles, path)
		return nil
	}
	for _, dir := range scenarioDirs {
		_ = filepath.WalkDir(dir, loader)
	}
	contents := make([][]Scenario, len(scenarioFiles))
	errGroup := &errgroup.Group{}
	for i, file := range scenarioFiles {
		file := file
		errGroup.Go(func() error {
			b, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			if strings.HasPrefix(strings.TrimSpace(string(b)), "[") {
				var scenarios []Scenario
				if err := json.Unmarshal(b, &scenarios); err != nil {
					return err
				}
				contents[i] = scenarios
			} else {
				var scenario Scenario
				if err := json.Unmarshal(b, &scenario); err != nil {
					return err
				}
				contents[i] = []Scenario{scenario}
			}
			return nil
		})
	}
	if err := errGroup.Wait(); err != nil {
		return nil, err
	}
	for _, scenarios := range contents {
		allScenarios = append(allScenarios, scenarios...)
	}
	return allScenarios, nil
}

func updateSelectorCache(repo *policy.Repository, allocator *cache.CachingIdentityAllocator) {
	wg := sync.WaitGroup{}
	selectorCache := repo.GetSelectorCache()
	added := allocator.GetIdentityCache()
	selectorCache.UpdateIdentities(added, nil, &wg)
	wg.Wait()
}

func RunScenarios(logger *slog.Logger, repo *policy.Repository, idm identitymanager.IDManager, scenarios []Scenario, isVerbose bool) []ScenarioResult {
	results := make([]ScenarioResult, 0, len(scenarios))
	resultsChannel := make(chan ScenarioResult, len(scenarios))
	wg := sync.WaitGroup{}
	wg.Add(len(scenarios))
	for _, scenario := range scenarios {
		go func(scenario Scenario) {
			defer wg.Done()
			db := statedb.New()
			db.Start()
			defer db.Stop()
			allocatorLogger := logger
			if !isVerbose {
				allocatorLogger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
			}
			allocator := getMockIdentityAllocator(allocatorLogger, db)
			flow := scenario.ToFlow(allocator)
			if flow == nil {
				slog.Error("invalid flow", "scenario", scenario)
				return
			}
			updateSelectorCache(repo, allocator)
			srcEP, destEP := scenario.ToEndpointInfo(logger, flow)
			decision, _, _, err := policy.LookupFlow(logger, repo, idm, *flow, srcEP, destEP)
			if err != nil {
				slog.Error("error looking up flow", "scenario", scenario, "error", err)
				return
			}
			result := ScenarioResult{
				Name:     scenario.Name,
				Expected: scenario.ExpectedVerdict,
				Actual:   ParseDecision(decision),
			}
			resultsChannel <- result
		}(scenario)
	}
	wg.Wait()
	close(resultsChannel)
	for result := range resultsChannel {
		results = append(results, result)
	}

	return results
}

func OutputResults(results []ScenarioResult, w io.Writer, isJSON bool) error {
	slices.SortFunc(results, func(a, b ScenarioResult) int {
		return strings.Compare(a.Name, b.Name)
	})
	if isJSON {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(results)
	}

	for _, result := range results {
		_, err := w.Write([]byte(result.Name + ": " + result.Expected.String() + " -> " + result.Actual.String() + "\n"))
		if err != nil {
			return err
		}
	}
	return nil
}
