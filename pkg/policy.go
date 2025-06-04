package pkg

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/cilium/cilium/pkg/policy/api"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// LoadPolicies loads Cilium Network Policies from the specified directories and files.
func LoadPolicies(policyDirs, policyFiles []string) ([]*api.Rule, error) {
	var allRules []*api.Rule
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
	for _, file := range policyFiles {
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		decoder := yaml.NewYAMLOrJSONDecoder(f, 4096)
		for {
			var obj unstructured.Unstructured
			if err := decoder.Decode(&obj); err != nil {
				if err != io.EOF {
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
				rules, err := cnp.Parse()
				if err != nil {
					return nil, err
				}
				allRules = append(allRules, rules...)
			case "CiliumClusterwideNetworkPolicy":
				var ccnp v2.CiliumClusterwideNetworkPolicy
				if err := yaml.Unmarshal(jsonData, &ccnp); err != nil {
					return nil, err
				}
				rules, err := ccnp.Parse()
				if err != nil {
					return nil, err
				}
				allRules = append(allRules, rules...)
			}
		}
	}
	return allRules, nil
}
