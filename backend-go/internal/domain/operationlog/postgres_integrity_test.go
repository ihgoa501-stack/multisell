package operationlog

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPostgresAuditChainConcurrentInserts(t *testing.T) {
	dsn := os.Getenv("OPERATIONLOG_POSTGRES_DESTRUCTIVE_TEST_DSN")
	if dsn == "" {
		t.Skip("set OPERATIONLOG_POSTGRES_DESTRUCTIVE_TEST_DSN to a disposable database migrated through 000152")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("TRUNCATE operation_log RESTART IDENTITY").Error; err != nil {
		t.Fatal(err)
	}

	svc := NewService(db, zap.NewNop())
	const writes = 64
	start := make(chan struct{})
	errs := make(chan error, writes)
	var wg sync.WaitGroup
	for i := 0; i < writes; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs <- svc.Create(&OperationLog{Module: "concurrency", Action: fmt.Sprintf("write_%d", i)})
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	if err := svc.VerifyIntegrity(context.Background()); err != nil {
		t.Fatal(err)
	}
}
