package config

import "testing"

func TestLoadNoConfigIsNotAnError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, ok, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ok {
		t.Fatalf("expected no config on first run, got %+v", cfg)
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	want := Config{DataDir: "/tmp/whatever/focuson-data"}
	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, ok, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok || got.DataDir != want.DataDir {
		t.Fatalf("round trip mismatch: got %+v ok=%v, want %+v", got, ok, want)
	}
}
