package imagegen

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &ProductImageGen{}, &ProductCanvas{}, &PromptTemplate{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

func seedImageGen(t *testing.T, svc *Service, productID int64) *ProductImageGen {
	t.Helper()
	g := &ProductImageGen{
		ProductID: productID,
		Prompt:    "a white background product photo",
		Style:     "product_white",
		Status:    "pending",
	}
	if err := svc.CreateImageGen(g); err != nil {
		t.Fatalf("seedImageGen failed: %v", err)
	}
	return g
}

func seedTemplate(t *testing.T, svc *Service) *PromptTemplate {
	t.Helper()
	tpl := &PromptTemplate{
		Name:     "Product Photo",
		Prompt:   "professional photo of {{product}}",
		Style:    "product_white",
		IsShared: 1,
	}
	if err := svc.CreateTemplate(tpl); err != nil {
		t.Fatalf("seedTemplate failed: %v", err)
	}
	return tpl
}

func TestImageGen_Create(t *testing.T) {
	svc := newService(t)

	g := &ProductImageGen{
		ProductID: 1,
		Prompt:    "a white background product photo",
		Style:     "product_white",
		Status:    "pending",
	}
	if err := svc.CreateImageGen(g); err != nil {
		t.Fatalf("CreateImageGen failed: %v", err)
	}
	if g.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestImageGen_GetByID(t *testing.T) {
	svc := newService(t)

	created := seedImageGen(t, svc, 42)

	got, err := svc.GetImageGen(created.ID)
	if err != nil {
		t.Fatalf("GetImageGen failed: %v", err)
	}
	if got.ProductID != 42 {
		t.Fatalf("ProductID = %d, want 42", got.ProductID)
	}
	if got.Prompt != "a white background product photo" {
		t.Fatalf("Prompt = %q, want correct value", got.Prompt)
	}
}

func TestImageGen_GetByID_NotFound(t *testing.T) {
	svc := newService(t)

	_, err := svc.GetImageGen(999)
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestImageGen_List_FilterByProduct(t *testing.T) {
	svc := newService(t)

	seedImageGen(t, svc, 1)
	seedImageGen(t, svc, 1)
	seedImageGen(t, svc, 2)

	items, total, err := svc.ListImageGens(1, "", "", 1, 20)
	if err != nil {
		t.Fatalf("ListImageGens failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("returned %d items, want 2", len(items))
	}
}

func TestImageGen_List_Pagination(t *testing.T) {
	svc := newService(t)

	for i := 0; i < 15; i++ {
		seedImageGen(t, svc, 100)
	}

	items, total, err := svc.ListImageGens(100, "", "", 1, 5)
	if err != nil {
		t.Fatalf("ListImageGens failed: %v", err)
	}
	if total != 15 {
		t.Fatalf("total = %d, want 15", total)
	}
	if len(items) != 5 {
		t.Fatalf("returned %d items, want 5", len(items))
	}

	items2, _, err := svc.ListImageGens(100, "", "", 2, 5)
	if err != nil {
		t.Fatalf("ListImageGens page 2 failed: %v", err)
	}
	if len(items2) != 5 {
		t.Fatalf("page 2 returned %d items, want 5", len(items2))
	}
	if items[0].ID == items2[0].ID {
		t.Fatal("page 1 and page 2 should not start with the same record")
	}
}

func TestImageGen_UpdateStatus(t *testing.T) {
	svc := newService(t)

	g := seedImageGen(t, svc, 10)

	if err := svc.UpdateImageGenStatus(g.ID, "completed", []byte(`["url1","url2"]`), ""); err != nil {
		t.Fatalf("UpdateImageGenStatus failed: %v", err)
	}

	got, err := svc.GetImageGen(g.ID)
	if err != nil {
		t.Fatalf("GetImageGen failed: %v", err)
	}
	if got.Status != "completed" {
		t.Fatalf("Status = %q, want completed", got.Status)
	}
}

func TestImageGen_UpdateStatus_Error(t *testing.T) {
	svc := newService(t)

	g := seedImageGen(t, svc, 10)

	if err := svc.UpdateImageGenStatus(g.ID, "failed", nil, "API timeout"); err != nil {
		t.Fatalf("UpdateImageGenStatus failed: %v", err)
	}

	got, err := svc.GetImageGen(g.ID)
	if err != nil {
		t.Fatalf("GetImageGen failed: %v", err)
	}
	if got.ErrorMessage != "API timeout" {
		t.Fatalf("ErrorMessage = %q, want API timeout", got.ErrorMessage)
	}
}

func TestImageGen_Delete(t *testing.T) {
	svc := newService(t)

	g := seedImageGen(t, svc, 5)

	if err := svc.DeleteImageGen(g.ID); err != nil {
		t.Fatalf("DeleteImageGen failed: %v", err)
	}

	_, err := svc.GetImageGen(g.ID)
	if err == nil {
		t.Fatal("expected error after DeleteImageGen")
	}
}

func TestImageGen_Delete_NotFound(t *testing.T) {
	svc := newService(t)

	if err := svc.DeleteImageGen(999); err != nil {
		t.Fatalf("DeleteImageGen for non-existent ID should succeed: %v", err)
	}
}

func TestTemplate_Create(t *testing.T) {
	svc := newService(t)

	tpl := &PromptTemplate{
		Name:     "Lifestyle Shot",
		Prompt:   "product in lifestyle setting",
		Style:    "lifestyle",
		IsShared: 1,
	}
	if err := svc.CreateTemplate(tpl); err != nil {
		t.Fatalf("CreateTemplate failed: %v", err)
	}
	if tpl.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestTemplate_GetByID(t *testing.T) {
	svc := newService(t)

	created := seedTemplate(t, svc)

	got, err := svc.GetTemplate(created.ID)
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}
	if got.Name != "Product Photo" {
		t.Fatalf("Name = %q, want Product Photo", got.Name)
	}
}

func TestTemplate_GetByID_NotFound(t *testing.T) {
	svc := newService(t)

	_, err := svc.GetTemplate(999)
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestTemplate_List(t *testing.T) {
	svc := newService(t)

	seedTemplate(t, svc)
	seedTemplate(t, svc)

	items, total, err := svc.ListTemplates("", "", 0, 1, 20)
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("returned %d items, want 2", len(items))
	}
}

func TestTemplate_Update(t *testing.T) {
	svc := newService(t)

	tpl := seedTemplate(t, svc)
	tpl.Name = "Updated Template"
	tpl.Prompt = "updated prompt text"

	if err := svc.UpdateTemplate(tpl); err != nil {
		t.Fatalf("UpdateTemplate failed: %v", err)
	}

	got, err := svc.GetTemplate(tpl.ID)
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}
	if got.Name != "Updated Template" {
		t.Fatalf("Name = %q, want Updated Template", got.Name)
	}
	if got.Prompt != "updated prompt text" {
		t.Fatalf("Prompt = %q, want updated prompt text", got.Prompt)
	}
}

func TestTemplate_IncrementUsage(t *testing.T) {
	svc := newService(t)

	tpl := seedTemplate(t, svc)
	if tpl.UsageCount != 0 {
		t.Fatalf("initial UsageCount = %d, want 0", tpl.UsageCount)
	}

	if err := svc.IncrementTemplateUsage(tpl.ID); err != nil {
		t.Fatalf("IncrementTemplateUsage failed: %v", err)
	}
	if err := svc.IncrementTemplateUsage(tpl.ID); err != nil {
		t.Fatalf("IncrementTemplateUsage failed: %v", err)
	}

	got, err := svc.GetTemplate(tpl.ID)
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}
	if got.UsageCount != 2 {
		t.Fatalf("UsageCount = %d, want 2", got.UsageCount)
	}
}
