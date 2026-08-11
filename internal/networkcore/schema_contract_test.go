package networkcore

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var networkSamplePaths = []string{
	"samples/network/node-china-egress.synthetic.json",
	"samples/network/node-global-egress.synthetic.json",
	"samples/network/targets-domestic.synthetic.json",
	"samples/network/targets-global.synthetic.json",
	"samples/network/protected-management.synthetic.json",
	"samples/network/profile-overseas-host.synthetic.json",
	"samples/network/profile-domestic-host.synthetic.json",
}

func TestPublicNetworkSamplesMatchStrictContracts(t *testing.T) {
	root := networkRepositoryRoot(t)
	tests := []struct {
		path   string
		decode func(io.Reader) error
	}{
		{"samples/network/node-china-egress.synthetic.json", func(reader io.Reader) error { _, err := DecodeNode(reader); return err }},
		{"samples/network/node-global-egress.synthetic.json", func(reader io.Reader) error { _, err := DecodeNode(reader); return err }},
		{"samples/network/targets-domestic.synthetic.json", func(reader io.Reader) error { _, err := DecodeTargetSet(reader); return err }},
		{"samples/network/targets-global.synthetic.json", func(reader io.Reader) error { _, err := DecodeTargetSet(reader); return err }},
		{"samples/network/protected-management.synthetic.json", func(reader io.Reader) error { _, err := DecodeTargetSet(reader); return err }},
		{"samples/network/profile-overseas-host.synthetic.json", func(reader io.Reader) error { _, err := DecodeRouteProfile(reader); return err }},
		{"samples/network/profile-domestic-host.synthetic.json", func(reader io.Reader) error { _, err := DecodeRouteProfile(reader); return err }},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			file, err := os.Open(filepath.Join(root, filepath.FromSlash(test.path)))
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			if err := test.decode(file); err != nil {
				t.Fatalf("strict decode failed: %v", err)
			}
		})
	}
}

func TestPublicNetworkSamplesContainNoSecretMaterial(t *testing.T) {
	root := networkRepositoryRoot(t)
	for _, path := range networkSamplePaths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		lower := strings.ToLower(string(data))
		for _, forbidden := range []string{"password", "private_key", "subscription_url", "access_token", "bearer "} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s contains %q", path, forbidden)
			}
		}
	}
}

func networkRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}
