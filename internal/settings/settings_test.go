package settings

import "testing"

func TestCompleteSetupPersists(t *testing.T) {
	root := t.TempDir()
	st, err := NewStore(root).Get()
	if err != nil || st.SetupCompleted {
		t.Fatalf("fresh store: %+v, %v", st, err)
	}
	if _, err := NewStore(root).CompleteSetup(); err != nil {
		t.Fatal(err)
	}
	if st, _ := NewStore(root).Get(); !st.SetupCompleted {
		t.Fatal("setup flag not persisted")
	}
}

func TestUpdateKeepsSetupFlag(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.CompleteSetup(); err != nil {
		t.Fatal(err)
	}
	st, err := store.Update(Settings{AllowNSFW: true})
	if err != nil {
		t.Fatal(err)
	}
	if !st.SetupCompleted || !st.AllowNSFW {
		t.Fatalf("after update: %+v", st)
	}
}
