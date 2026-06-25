package finance

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &FinanceAccount{}, &FinanceTransaction{}, &FinanceLedgerEntry{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

// ── Account CRUD ─────────────────────────────────────────────────────

func TestAccount_Create_Defaults(t *testing.T) {
	svc := newService(t)

	in := &CreateAccountInput{Name: "Test Bank", AccountType: "bank"}
	a, err := svc.CreateAccount(in)
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if a.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if a.Currency != "CNY" {
		t.Fatalf("Currency=%q, want CNY", a.Currency)
	}
	if a.Status != "active" {
		t.Fatalf("Status=%q, want active", a.Status)
	}
}

func TestAccount_Create_WithBalance(t *testing.T) {
	svc := newService(t)

	bal := 5000.50
	in := &CreateAccountInput{Name: "Alipay", AccountType: "payment", Balance: &bal}
	a, err := svc.CreateAccount(in)
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if a.Balance != 5000.50 {
		t.Fatalf("Balance=%f, want 5000.50", a.Balance)
	}
}

func TestAccount_GetByID(t *testing.T) {
	svc := newService(t)

	created, _ := svc.CreateAccount(&CreateAccountInput{Name: "Findable", AccountType: "bank"})
	got, err := svc.GetAccount(created.ID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.Name != "Findable" {
		t.Fatalf("Name=%q, want Findable", got.Name)
	}
}

func TestAccount_Update(t *testing.T) {
	svc := newService(t)

	created, _ := svc.CreateAccount(&CreateAccountInput{Name: "Old Name", AccountType: "bank"})
	newName := "New Name"
	updated, err := svc.UpdateAccount(created.ID, &UpdateAccountInput{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	if updated.Name != "New Name" {
		t.Fatalf("Name=%q, want New Name", updated.Name)
	}
}

func TestAccount_Delete(t *testing.T) {
	svc := newService(t)

	created, _ := svc.CreateAccount(&CreateAccountInput{Name: "Doomed", AccountType: "bank"})
	if err := svc.DeleteAccount(created.ID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	_, err := svc.GetAccount(created.ID)
	if err == nil {
		t.Fatal("expected error for deleted account")
	}
}

func TestAccount_Delete_NotFound(t *testing.T) {
	svc := newService(t)

	err := svc.DeleteAccount(99999)
	if err == nil {
		t.Fatal("expected ErrRecordNotFound for missing account")
	}
}

func TestAccount_ListAccounts_FilterByType(t *testing.T) {
	svc := newService(t)

	svc.CreateAccount(&CreateAccountInput{Name: "A1", AccountType: "bank"})
	svc.CreateAccount(&CreateAccountInput{Name: "A2", AccountType: "payment"})
	svc.CreateAccount(&CreateAccountInput{Name: "A3", AccountType: "bank"})

	p := common.Pagination{Page: 1, Size: 20}
	f := &AccountListFilter{AccountType: "bank"}
	items, total, err := svc.ListAccounts(&p, f)
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len=%d, want 2", len(items))
	}
}

// ── Transactions ──────────────────────────────────────────────────────

func TestCreateTransaction_Revenue_IncreasesBalance(t *testing.T) {
	svc := newService(t)

	a, _ := svc.CreateAccount(&CreateAccountInput{Name: "Rev Account", AccountType: "platform"})
	if a.Balance != 0 {
		t.Fatalf("initial balance=%f, want 0", a.Balance)
	}

	_, err := svc.CreateTransaction(&CreateTransactionInput{
		AccountID:       a.ID,
		TransactionType: "revenue",
		Amount:          1000,
	})
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}

	got, _ := svc.GetAccount(a.ID)
	if got.Balance != 1000 {
		t.Fatalf("balance after revenue=%f, want 1000", got.Balance)
	}
}

func TestCreateTransaction_Cost_DecreasesBalance(t *testing.T) {
	svc := newService(t)

	a, _ := svc.CreateAccount(&CreateAccountInput{Name: "Cost Account", AccountType: "bank"})
	// Seed with balance
	svc.db.Model(&FinanceAccount{}).Where("id = ?", a.ID).Update("balance", 5000)

	_, err := svc.CreateTransaction(&CreateTransactionInput{
		AccountID:       a.ID,
		TransactionType: "cost",
		Amount:          1200,
	})
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}

	got, _ := svc.GetAccount(a.ID)
	if got.Balance != 3800 {
		t.Fatalf("balance after cost=%f, want 3800", got.Balance)
	}
}

func TestCreateTransaction_Fee_DecreasesBalance(t *testing.T) {
	svc := newService(t)

	a, _ := svc.CreateAccount(&CreateAccountInput{Name: "Fee Account", AccountType: "bank"})
	svc.db.Model(&FinanceAccount{}).Where("id = ?", a.ID).Update("balance", 2000)

	_, err := svc.CreateTransaction(&CreateTransactionInput{
		AccountID:       a.ID,
		TransactionType: "fee",
		Amount:          50,
	})
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}

	got, _ := svc.GetAccount(a.ID)
	if got.Balance != 1950 {
		t.Fatalf("balance after fee=%f, want 1950", got.Balance)
	}
}

func TestListTransactions_FilterByAccountID(t *testing.T) {
	svc := newService(t)

	a1, _ := svc.CreateAccount(&CreateAccountInput{Name: "Acc1", AccountType: "bank"})
	a2, _ := svc.CreateAccount(&CreateAccountInput{Name: "Acc2", AccountType: "bank"})

	svc.CreateTransaction(&CreateTransactionInput{AccountID: a1.ID, TransactionType: "revenue", Amount: 100})
	svc.CreateTransaction(&CreateTransactionInput{AccountID: a2.ID, TransactionType: "revenue", Amount: 200})
	svc.CreateTransaction(&CreateTransactionInput{AccountID: a1.ID, TransactionType: "cost", Amount: 30})

	p := common.Pagination{Page: 1, Size: 20}
	f := &TransactionListFilter{AccountID: &a1.ID}
	items, total, err := svc.ListTransactions(&p, f)
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len=%d, want 2", len(items))
	}
}

func TestListTransactions_FilterByType(t *testing.T) {
	svc := newService(t)

	a, _ := svc.CreateAccount(&CreateAccountInput{Name: "X", AccountType: "bank"})
	svc.CreateTransaction(&CreateTransactionInput{AccountID: a.ID, TransactionType: "revenue", Amount: 100})
	svc.CreateTransaction(&CreateTransactionInput{AccountID: a.ID, TransactionType: "revenue", Amount: 200})
	svc.CreateTransaction(&CreateTransactionInput{AccountID: a.ID, TransactionType: "cost", Amount: 50})

	p := common.Pagination{Page: 1, Size: 20}
	f := &TransactionListFilter{TransactionType: "cost"}
	items, total, err := svc.ListTransactions(&p, f)
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if total != 1 {
		t.Fatalf("total=%d, want 1", total)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d, want 1", len(items))
	}
}

// ── Ledger Entries ────────────────────────────────────────────────────

func TestListLedgerEntries_FilterByEntryType(t *testing.T) {
	svc := newService(t)

	orderID := int64(1)
	svc.db.Create(&FinanceLedgerEntry{OrderID: &orderID, EntryType: "revenue", Amount: 500, Currency: "CNY"})
	svc.db.Create(&FinanceLedgerEntry{OrderID: &orderID, EntryType: "product_cost", Amount: 200, Currency: "CNY"})
	svc.db.Create(&FinanceLedgerEntry{OrderID: &orderID, EntryType: "revenue", Amount: 300, Currency: "CNY"})

	p := common.Pagination{Page: 1, Size: 20}
	f := &LedgerListFilter{EntryType: "revenue"}
	items, total, err := svc.ListLedgerEntries(&p, f)
	if err != nil {
		t.Fatalf("ListLedgerEntries: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len=%d, want 2", len(items))
	}
}

// ── Summary ───────────────────────────────────────────────────────────

func TestSummary_WithData(t *testing.T) {
	svc := newService(t)

	svc.CreateAccount(&CreateAccountInput{Name: "A1", AccountType: "bank"})
	svc.CreateAccount(&CreateAccountInput{Name: "A2", AccountType: "payment"})

	oid1 := int64(1)
	oid2 := int64(2)
	svc.db.Create(&FinanceLedgerEntry{OrderID: &oid1, EntryType: "revenue", Amount: 1000, Currency: "CNY"})
	svc.db.Create(&FinanceLedgerEntry{OrderID: &oid2, EntryType: "product_cost", Amount: 400, Currency: "CNY"})

	sum, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if sum.TotalBalance != 0 {
		t.Fatalf("TotalBalance=%f, want 0", sum.TotalBalance)
	}
	if len(sum.LedgerByEntryType) != 2 {
		t.Fatalf("LedgerByEntryType len=%d, want 2", len(sum.LedgerByEntryType))
	}
	if sum.LedgerByEntryType["revenue"] != 1000 {
		t.Fatalf("revenue=%f, want 1000", sum.LedgerByEntryType["revenue"])
	}
	if sum.LedgerByEntryType["product_cost"] != 400 {
		t.Fatalf("product_cost=%f, want 400", sum.LedgerByEntryType["product_cost"])
	}
}

// ── Mock ──────────────────────────────────────────────────────────────

func TestMock_GeneratesCorrectCount(t *testing.T) {
	svc := newService(t)

	entries, err := svc.Mock(5)
	if err != nil {
		t.Fatalf("Mock: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("len=%d, want 5", len(entries))
	}

	var count int64
	svc.db.Model(&FinanceLedgerEntry{}).Count(&count)
	if count != 5 {
		t.Fatalf("db count=%d, want 5", count)
	}
}

func TestMock_DefaultsTo10(t *testing.T) {
	svc := newService(t)

	entries, err := svc.Mock(0)
	if err != nil {
		t.Fatalf("Mock: %v", err)
	}
	if len(entries) != 10 {
		t.Fatalf("len=%d, want 10 (default)", len(entries))
	}
}
