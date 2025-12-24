package pkg

import (
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/cilium/cilium/pkg/policy/api"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func loadPoliciesFromFile(logger *slog.Logger, file string) (api.Rules, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := yaml.NewYAMLOrJSONDecoder(f, 4096)
	var allRules api.Rules
	for {
		var obj unstructured.Unstructured
		if err := decoder.Decode(&obj); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		kind := obj.GetKind()
		jsonData, err := obj.MarshalJSON()
		if err != nil {
			return nil, err
		}
		switch kind {
		case "CiliumNetworkPolicy":
			var cnp v2.CiliumNetworkPolicy
			if err := yaml.Unmarshal(jsonData, &cnp); err != nil {
				return nil, err
			}
			rules, err := cnp.Parse(logger, "")
			if err != nil {
				return nil, err
			}
			allRules = append(allRules, rules...)
		case "CiliumClusterwideNetworkPolicy":
			var ccnp v2.CiliumClusterwideNetworkPolicy
			if err := yaml.Unmarshal(jsonData, &ccnp); err != nil {
				return nil, err
			}
			rules, err := ccnp.Parse(logger, "")
			if err != nil {
				return nil, err
			}
			allRules = append(allRules, rules...)
		default:
			continue // Skip unsupported kinds
		}
	}
	return allRules, nil
}

// LoadPolicies loads Cilium Network Policies from the specified directories and files.
func LoadPolicies(logger *slog.Logger, policyDirs, policyFiles []string) (api.Rules, error) {
	var allRules api.Rules
	loader := func (path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}
		policyFiles = append(policyFiles, path)
		return nil
	}
	for _, dir := range policyDirs {
		_ = filepath.WalkDir(dir, loader)
	}
	contents := make([]api.Rules, len(policyFiles))
	errGroup := &errgroup.Group{}
	for i, file := range policyFiles {
		file := file
		errGroup.Go(func() error {
			rules, err := loadPoliciesFromFile(logger, file)
			if err != nil {
				return err
			}
			contents[i] = rules
			return nil
		})
	}
	if err := errGroup.Wait(); err != nil {
		return nil, err
	}
	for _, rules := range contents {
		allRules = append(allRules, rules...)
	}
	return allRules, nil
}
