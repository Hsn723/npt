package pkg

import (
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/cilium/cilium/api/v1/models"
	"github.com/cilium/cilium/pkg/labels"
	"github.com/cilium/cilium/pkg/policy"
	"github.com/cilium/cilium/pkg/policy/api"
	"golang.org/x/sync/errgroup"
)

type Scenario struct {
	Name            string         `json:"name"`
	From            []labels.Label `json:"from"`
	To              []labels.Label `json:"to"`
	DPorts          []*models.Port `json:"dPorts"`
	Direction       Direction      `json:"direction"`
	ExpectedVerdict Verdict        `json:"expectedVerdict"`
}

func (s Scenario) ToSearchContext(verbose bool) *policy.SearchContext {
	sc := &policy.SearchContext{
		From:   s.From,
		To:     s.To,
		DPorts: s.DPorts,
	}
	if verbose {
		sc.Logging = log.New(os.Stderr, "", 0)
		sc.Trace = policy.TRACE_VERBOSE
	}
	return sc
}

type Direction string

const (
	DirectionIngress Direction = "ingress"
	DirectionEgress  Direction = "egress"
)

type Verdict string

const (
	VercictAllow     Verdict = "Allowed"
	VerdictDeny      Verdict = "Denied"
	VerdictUndecided Verdict = "Undecided"
)

func (v Verdict) Decision() api.Decision {
	switch v {
	case VercictAllow:
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
		return VercictAllow
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

func RunScenarios(repo *policy.Repository, scenarios []Scenario, isVerbose bool) []ScenarioResult {
	results := make([]ScenarioResult, 0, len(scenarios))
	resultsChannel := make(chan ScenarioResult, len(scenarios))
	wg := sync.WaitGroup{}
	wg.Add(len(scenarios))
	for _, scenario := range scenarios {
		go func(scenario Scenario) {
			searchContext := scenario.ToSearchContext(isVerbose)
			var decision api.Decision
			switch scenario.Direction {
			case DirectionIngress:
				decision = repo.AllowsIngressRLocked(searchContext)
			case DirectionEgress:
				decision = repo.AllowsEgressRLocked(searchContext)
			}
			result := ScenarioResult{
				Name:     scenario.Name,
				Expected: scenario.ExpectedVerdict,
				Actual:   ParseDecision(decision),
			}
			resultsChannel <- result
			wg.Done()
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
