package tunrepair

import "testing"

func TestChooseBasePortMovesAwayFromExcludedControllerRange(t *testing.T) {
	t.Parallel()

	const currentBase = 12000
	excluded := []PortRange{{Start: currentBase + 6, End: currentBase + 20}}

	selected, changed, err := ChooseBasePort(
		currentBase,
		excluded,
		nil,
		20000,
		20100,
	)
	if err != nil {
		t.Fatalf("ChooseBasePort() error = %v", err)
	}
	if !changed {
		t.Fatal("ChooseBasePort() changed = false, want true")
	}
	if selected < 20000 || selected > 20100 {
		t.Fatalf("selected base = %d, want inside scan range", selected)
	}
}

func TestChooseBasePortKeepsUsableCurrentBase(t *testing.T) {
	t.Parallel()

	selected, changed, err := ChooseBasePort(
		22000,
		[]PortRange{{Start: 30000, End: 30100}},
		[]Listener{{Port: 31000}},
		20000,
		45000,
	)
	if err != nil {
		t.Fatalf("ChooseBasePort() error = %v", err)
	}
	if changed || selected != 22000 {
		t.Fatalf("selected = %d, changed = %v; want current base unchanged", selected, changed)
	}
}

func TestSetConfigLocalPortChangesOnlyFirstInboundBase(t *testing.T) {
	t.Parallel()

	input := []byte(`{
  "TunModeItem": {"EnableTun": true},
  "Inbound": [
    {"LocalPort": 12000, "Protocol": "socks"},
    {"LocalPort": 13000, "Protocol": "http"}
  ]
}`)

	updated, changed, err := SetConfigLocalPort(input, 24000)
	if err != nil {
		t.Fatalf("SetConfigLocalPort() error = %v", err)
	}
	if !changed {
		t.Fatal("SetConfigLocalPort() changed = false, want true")
	}
	port, err := ConfigLocalPort(updated)
	if err != nil {
		t.Fatalf("ConfigLocalPort(updated) error = %v", err)
	}
	if port != 24000 {
		t.Fatalf("updated first inbound port = %d, want 24000", port)
	}
	if string(updated) == string(input) {
		t.Fatal("updated config is byte-identical to input")
	}
}
