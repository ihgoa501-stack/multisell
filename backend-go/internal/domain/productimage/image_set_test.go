package productimage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func imageSetTestDB(t *testing.T) *ImageSetService {
	t.Helper()
	db := dbtest.NewDB(t, &ImageSet{}, &ImageSetItem{})
	return NewImageSetService(db)
}

func validImageSetInput() CreateImageSetInput {
	manifest := fmt.Sprintf("%064x", 9)
	return CreateImageSetInput{
		OwnerID: 7, ListingID: 42, Channel: "Amazon-US", Locale: "en-US",
		Items: []ImageSetItemInput{
			{Role: "main", Ordinal: 1, Locale: "en-US", Channel: "amazon-us", AssetSHA: fmt.Sprintf("%064x", 1), TaskID: 1, OutputBlobID: fmt.Sprintf("%064x", 1), TaskManifestHash: manifest, Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", ImageServiceJobID: "job-1"},
			{Role: "gallery", Ordinal: 2, Locale: "en-US", Channel: "amazon-us", AssetSHA: fmt.Sprintf("%064x", 2), TaskID: 2, OutputBlobID: fmt.Sprintf("%064x", 2), TaskManifestHash: manifest, Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", ImageServiceJobID: "job-2"},
		},
	}
}

func TestImageSetOwnerSelectFreezeAndRevision(t *testing.T) {
	svc := imageSetTestDB(t)
	ctx := context.Background()

	draft, err := svc.CreateDraft(ctx, validImageSetInput())
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if draft.Version != 1 || draft.Status != ImageSetDraft {
		t.Fatalf("unexpected draft: %#v", draft)
	}

	frozen, err := svc.SelectAndFreeze(ctx, 7, draft.ID)
	if err != nil {
		t.Fatalf("freeze: %v", err)
	}
	if frozen.Status != ImageSetFrozen || len(frozen.ManifestSHA) != 64 {
		t.Fatalf("unexpected frozen set: %#v", frozen)
	}
	if frozen.SelectedBy == nil || *frozen.SelectedBy != 7 || frozen.SelectedAt == nil || frozen.FrozenAt == nil {
		t.Fatal("Owner selection evidence was not recorded")
	}

	// Replaying an already-consumed Owner selection is safe and idempotent.
	replayed, err := svc.SelectAndFreeze(ctx, 7, draft.ID)
	if err != nil || replayed.ManifestSHA != frozen.ManifestSHA {
		t.Fatalf("idempotent freeze: %#v, %v", replayed, err)
	}

	changed := validImageSetInput().Items
	changed[1].AssetSHA = fmt.Sprintf("%064x", 3)
	changed[1].OutputBlobID = changed[1].AssetSHA
	revision, err := svc.Revise(ctx, 7, draft.ID, changed)
	if err != nil {
		t.Fatalf("revise: %v", err)
	}
	if revision.Version != 2 || revision.Status != ImageSetDraft || revision.BasedOnSetID == nil || *revision.BasedOnSetID != draft.ID {
		t.Fatalf("unexpected revision: %#v", revision)
	}

	original, err := svc.Get(ctx, 7, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if original.Status != ImageSetFrozen || original.Items[1].AssetSHA != fmt.Sprintf("%064x", 2) || original.ManifestSHA != frozen.ManifestSHA {
		t.Fatalf("revision mutated frozen version: %#v", original)
	}
	if _, err := svc.Revise(ctx, 8, draft.ID, changed); err == nil {
		t.Fatal("another Owner must not revise the set")
	}
}

func TestCanonicalImageSetManifestIgnoresRowAndInputOrder(t *testing.T) {
	baseIn := validImageSetInput()
	base := ImageSet{OwnerID: 1, ListingID: 2, Channel: "ozon", Locale: "ru-RU", Version: 3, Items: []ImageSetItem{
		{ID: 10, Role: "gallery", Ordinal: 2, Locale: "ru-RU", Channel: "ozon", AssetSHA: baseIn.Items[1].AssetSHA, TaskID: 2, OutputBlobID: baseIn.Items[1].OutputBlobID, TaskManifestHash: baseIn.Items[1].TaskManifestHash, Operation: baseIn.Items[1].Operation, Processor: baseIn.Items[1].Processor, ImageServiceJobID: "job-2"},
		{ID: 11, Role: "main", Ordinal: 1, Locale: "ru-RU", Channel: "ozon", AssetSHA: baseIn.Items[0].AssetSHA, TaskID: 1, OutputBlobID: baseIn.Items[0].OutputBlobID, TaskManifestHash: baseIn.Items[0].TaskManifestHash, Operation: baseIn.Items[0].Operation, Processor: baseIn.Items[0].Processor, ImageServiceJobID: "job-1"},
	}}
	first, err := CanonicalImageSetManifest(base)
	if err != nil {
		t.Fatal(err)
	}
	base.ID = 999
	base.Items[0].ID = 800
	base.Items[0], base.Items[1] = base.Items[1], base.Items[0]
	second, err := CanonicalImageSetManifest(base)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("canonical manifest changed: %s != %s", first, second)
	}

	base.Items[0].AssetSHA = fmt.Sprintf("%064x", 9)
	base.Items[0].OutputBlobID = base.Items[0].AssetSHA
	third, _ := CanonicalImageSetManifest(base)
	if third == first {
		t.Fatal("asset byte change must change manifest")
	}
}

func TestImageSetRoleOrdinalAndScopeConstraints(t *testing.T) {
	tests := []struct {
		name string
		edit func(*CreateImageSetInput)
	}{
		{"missing main", func(in *CreateImageSetInput) { in.Items[0].Role = "gallery" }},
		{"duplicate main", func(in *CreateImageSetInput) { in.Items[1].Role = "main" }},
		{"duplicate ordinal", func(in *CreateImageSetInput) { in.Items[1].Ordinal = 1 }},
		{"ordinal gap", func(in *CreateImageSetInput) { in.Items[1].Ordinal = 3 }},
		{"wrong channel", func(in *CreateImageSetInput) { in.Items[1].Channel = "ozon" }},
		{"wrong locale", func(in *CreateImageSetInput) { in.Items[1].Locale = "de-DE" }},
		{"bad hash", func(in *CreateImageSetInput) { in.Items[1].AssetSHA = "not-a-hash" }},
		{"unknown role", func(in *CreateImageSetInput) { in.Items[1].Role = "hero_magic" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := imageSetTestDB(t)
			in := validImageSetInput()
			tc.edit(&in)
			if _, err := svc.CreateDraft(context.Background(), in); !errors.Is(err, ErrImageSetInvalid) {
				t.Fatalf("got %v, want ErrImageSetInvalid", err)
			}
		})
	}
}

func TestImageSetConcurrentVersionsAreUnique(t *testing.T) {
	svc := imageSetTestDB(t)
	ctx := context.Background()
	const count = 12
	versions := make(chan uint, count)
	errs := make(chan error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			set, err := svc.CreateDraft(ctx, validImageSetInput())
			if err != nil {
				errs <- err
				return
			}
			versions <- set.Version
		}()
	}
	wg.Wait()
	close(errs)
	close(versions)
	for err := range errs {
		t.Fatalf("concurrent create: %v", err)
	}
	seen := make(map[uint]bool, count)
	for version := range versions {
		if seen[version] {
			t.Fatalf("duplicate version %d", version)
		}
		seen[version] = true
	}
	if len(seen) != count {
		t.Fatalf("got %d versions, want %d", len(seen), count)
	}
}
