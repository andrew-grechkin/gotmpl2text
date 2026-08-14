package uuid

import "testing"

func TestUUIDv7ToEpochNs(t *testing.T) {
	fm := FuncMap()
	fn, ok := fm["uuidv7ToEpochNs"].(func(string) (int64, error))
	if !ok {
		t.Fatalf("uuidv7ToEpochNs not registered or wrong signature")
	}

	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"valid UUID v7", "019e1c72-7449-7195-b65b-b7c7f94ed77e", 1778593723465000000, false},
		{"invalid UUID", "not-a-uuid", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fn(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestUUIDv7ToEpoch(t *testing.T) {
	fm := FuncMap()
	fn := fm["uuidv7ToEpoch"].(func(string) (int64, error))

	got, err := fn("019e1c72-7449-7195-b65b-b7c7f94ed77e")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 1778593723 {
		t.Errorf("got %d, want 1778593723", got)
	}

	if _, err := fn("bad-uuid"); err == nil {
		t.Errorf("expected error for invalid UUID")
	}
}

func TestUUIDv7Generate(t *testing.T) {
	fm := FuncMap()
	gen := fm["uuidv7"].(func() (string, error))
	toNs := fm["uuidv7ToEpochNs"].(func(string) (int64, error))

	id, err := gen()
	if err != nil {
		t.Fatalf("uuidv7: %v", err)
	}
	ns, err := toNs(id)
	if err != nil {
		t.Fatalf("generated uuid failed to parse: %v", err)
	}
	if ns < 1_000_000_000_000_000_000 {
		t.Errorf("timestamp too small: %d", ns)
	}
}
