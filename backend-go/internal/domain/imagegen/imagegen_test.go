package imagegen

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_ImageGen_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ProductImageGen{})
	svc := NewService(db, dbtest.NewLogger(t))

	g := &ProductImageGen{
		ProductID: 1,
		Prompt:    "a white background product photo",
		Style:     "product_white",
	}

	err := svc.CreateImageGen(g)
	if err != nil {
		t.Fatalf("CreateImageGen: %v", err)
	}
	if g.ID == 0 {
		t.Fatal("ID should be set")
	}
	if g.Status != "pending" {
		t.Fatalf("Status = %s (expected pending)", g.Status)
	}

	got, err := svc.GetImageGen(g.ID)
	if err != nil {
		t.Fatalf("GetImageGen: %v", err)
	}
	if got.ProductID != 1 {
		t.Fatalf("ProductID = %d", got.ProductID)
	}
}

func TestService_ImageGen_List(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ProductImageGen{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&ProductImageGen{ProductID: 1, Prompt: "p1", Style: "product_white", Status: "pending"})
	db.Create(&ProductImageGen{ProductID: 2, Prompt: "p2", Style: "product_white", Status: "completed"})
	db.Create(&ProductImageGen{ProductID: 1, Prompt: "p3", Style: "lifestyle", Status: "pending"})

	items, total, err := svc.ListImageGens(1, "", "", 1, 10)
	if err != nil {
		t.Fatalf("ListImageGens: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d (expected 2)", total)
	}
	_ = items
}

func TestService_ImageGen_UpdateStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ProductImageGen{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&ProductImageGen{ProductID: 1, Prompt: "test"})

	err := svc.UpdateImageGenStatus(1, "completed", []byte(`["https://img.url/1.jpg"]`), "")
	if err != nil {
		t.Fatalf("UpdateImageGenStatus: %v", err)
	}

	got, _ := svc.GetImageGen(1)
	if got.Status != "completed" {
		t.Fatalf("Status = %s", got.Status)
	}
}

func TestService_ImageGen_Delete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ProductImageGen{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&ProductImageGen{ProductID: 1, Prompt: "test"})
	if err := svc.DeleteImageGen(1); err != nil {
		t.Fatalf("DeleteImageGen: %v", err)
	}
	_, err := svc.GetImageGen(1)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

// ── ProductCanvas ──

func TestService_Canvas_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ProductCanvas{})
	svc := NewService(db, dbtest.NewLogger(t))

	cv := &ProductCanvas{ProductID: 1, Name: "主图画布"}
	err := svc.CreateCanvas(cv)
	if err != nil {
		t.Fatalf("CreateCanvas: %v", err)
	}
	if cv.ID == 0 {
		t.Fatal("ID should be set")
	}

	got, err := svc.GetCanvas(cv.ID)
	if err != nil {
		t.Fatalf("GetCanvas: %v", err)
	}
	if got.Name != "主图画布" {
		t.Fatalf("Name = %s", got.Name)
	}
}

func TestService_Canvas_UpdateAndDelete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ProductCanvas{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&ProductCanvas{ProductID: 1, Name: "旧画布"})

	items, total, err := svc.ListCanvases(1, 1, 10)
	if err != nil {
		t.Fatalf("ListCanvases: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	items[0].Name = "新画布"
	err = svc.UpdateCanvas(&items[0])
	if err != nil {
		t.Fatalf("UpdateCanvas: %v", err)
	}

	got, _ := svc.GetCanvas(items[0].ID)
	if got.Name != "新画布" {
		t.Fatalf("Name = %s", got.Name)
	}

	if err := svc.DeleteCanvas(items[0].ID); err != nil {
		t.Fatalf("DeleteCanvas: %v", err)
	}
}

// ── PromptTemplate ──

func TestService_Template_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PromptTemplate{})
	svc := NewService(db, dbtest.NewLogger(t))

	tpl := &PromptTemplate{
		Name:  "白色背景",
		Prompt: "A white background photo of {product}",
		Style: "product_white",
	}
	err := svc.CreateTemplate(tpl)
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	if tpl.ID == 0 {
		t.Fatal("ID should be set")
	}

	got, err := svc.GetTemplate(tpl.ID)
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if got.Name != "白色背景" {
		t.Fatalf("Name = %s", got.Name)
	}
}

func TestService_Template_ListAndIncrementUsage(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PromptTemplate{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&PromptTemplate{Name: "T1", Prompt: "p1", Style: "product_white", PlatformCode: "ozon", CreatedBy: 1})
	db.Create(&PromptTemplate{Name: "T2", Prompt: "p2", Style: "lifestyle", PlatformCode: "shopee", CreatedBy: 1})

	items, total, err := svc.ListTemplates("product_white", "", 0, 1, 10)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d (expected 1)", total)
	}
	_ = items

	// Increment usage
	if len(items) > 0 {
		err = svc.IncrementTemplateUsage(items[0].ID)
		if err != nil {
			t.Fatalf("IncrementTemplateUsage: %v", err)
		}
		got, _ := svc.GetTemplate(items[0].ID)
		if got.UsageCount != 1 {
			t.Fatalf("UsageCount = %d", got.UsageCount)
		}
	}
}

func TestService_Template_UpdateAndDelete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PromptTemplate{})
	svc := NewService(db, dbtest.NewLogger(t))

	tpl := &PromptTemplate{Name: "Old", Prompt: "old prompt"}
	svc.CreateTemplate(tpl)

	tpl.Prompt = "new prompt"
	err := svc.UpdateTemplate(tpl)
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}

	if err := svc.DeleteTemplate(tpl.ID); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	_, err = svc.GetTemplate(tpl.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
