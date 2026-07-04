package serverconfig

import "testing"

func TestSummarize(t *testing.T) {
	old := FromBytes("x.ini", []byte("PublicName=A\nMods=x\nWorkshopItems=1\nMap=Muldraugh, KY\n"))
	neu := FromBytes("x.ini", []byte("PublicName=B\nMods=lib;x\nWorkshopItems=1;2\nMap=Muldraugh, KY\n"))

	s := Summarize(old, neu)
	if s.Mods.Added != 1 {
		t.Fatalf("Mods.Added=%d want 1", s.Mods.Added)
	}
	if s.WorkshopItems.Added != 1 {
		t.Fatalf("WorkshopItems.Added=%d want 1", s.WorkshopItems.Added)
	}
	found := false
	for _, f := range s.ChangedFields {
		if f == "Server name" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'Server name' in ChangedFields, got %v", s.ChangedFields)
	}
	if s.Empty() {
		t.Fatal("summary should not be empty")
	}
}
