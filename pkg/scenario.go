package pkg

import (
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cilium/cilium/api/v1/models"
	"github.com/cilium/cilium/pkg/labels"
	"github.com/cilium/cilium/pkg/policy"
	"github.com/cilium/cilium/pkg/policy/api"
)

type Scenario struct {
	Name            string         `json:"name"`
	From            []labels.Label `json:"from"`
	To              []labels.Label `json:"to"`
	DPorts          []*models.Port `json:"dPorts"`
	Direction       Direction      `json:"direction"`
	ExpectedVerdict Verdict        `json:"expectedVerdict"`
}

func (s Scenario) ToSearchContext() *policy.SearchContext {
	return &policy.SearchContext{
		From:       s.From,
		To:         s.To,
		DPorts:     s.DPorts,
	}
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
	for _, file := range scenarioFiles {
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(strings.TrimSpace(string(b)), "[") {
			var scenarios []Scenario
			if err := json.Unmarshal(b, &scenarios); err != nil {
				return nil, err
			}
			allScenarios = append(allScenarios, scenarios...)
		} else {
			var scenario Scenario
			if err := json.Unmarshal(b, &scenario); err != nil {
				return nil, err
			}
			allScenarios = append(allScenarios, scenario)
		}
	}
	return allScenarios, nil
}

func RunScenarios(repo *policy.Repository, scenarios []Scenario) ([]ScenarioResult) {
	results := make([]ScenarioResult, 0, len(scenarios))
	for _, scenario := range scenarios {
		searchContext := scenario.ToSearchContext()
		var decision api.Decision
		switch scenario.Direction {
		case DirectionIngress:
			decision = repo.AllowsIngressRLocked(searchContext)
		case DirectionEgress:
			decision = repo.AllowsEgressRLocked(searchContext)
		}
		result := ScenarioResult{
			Name:           scenario.Name,
			Expected:       scenario.ExpectedVerdict,
			Actual:         ParseDecision(decision),
		}
		results = append(results, result)
	}

	return results
}

func OutputResults(results []ScenarioResult, w io.Writer, isJSON bool) error {
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
