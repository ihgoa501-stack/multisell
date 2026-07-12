package scheduler

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const schedulerAdvisoryLockID int64 = 61420260713

// LeaderLease serializes periodic scheduling across backend replicas. The
// returned release callback owns any resources that keep the lease alive.
type LeaderLease interface {
	TryAcquire(context.Context) (release func(context.Context) error, acquired bool, err error)
}

type PostgresLeaderLease struct{ db *gorm.DB }

func NewPostgresLeaderLease(db *gorm.DB) *PostgresLeaderLease { return &PostgresLeaderLease{db: db} }

func (l *PostgresLeaderLease) TryAcquire(ctx context.Context) (func(context.Context) error, bool, error) {
	sqlDB, err := l.db.DB()
	if err != nil {
		return nil, false, err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, false, err
	}
	var acquired bool
	if err := conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock($1)`, schedulerAdvisoryLockID).Scan(&acquired); err != nil {
		conn.Close()
		return nil, false, err
	}
	if !acquired {
		conn.Close()
		return nil, false, nil
	}
	release := func(releaseCtx context.Context) error {
		defer conn.Close()
		var unlocked bool
		if err := conn.QueryRowContext(releaseCtx, `SELECT pg_advisory_unlock($1)`, schedulerAdvisoryLockID).Scan(&unlocked); err != nil {
			return err
		}
		if !unlocked {
			return fmt.Errorf("scheduler: advisory lock was not held by lease connection")
		}
		return nil
	}
	return release, true, nil
}

func (s *Scheduler) acquireLeader(ctx context.Context) (func(context.Context) error, bool) {
	if s.leaderLease == nil {
		return func(context.Context) error { return nil }, true
	}
	retryInterval := s.leaderRetryInterval
	if retryInterval <= 0 {
		retryInterval = 5 * time.Second
	}
	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()
	for {
		release, acquired, err := s.leaderLease.TryAcquire(ctx)
		if err != nil {
			s.logger.Error("scheduler leader lease acquisition failed", zap.Error(err))
		} else if acquired {
			return release, true
		} else {
			s.logger.Info("scheduler standby waiting for leader lease")
		}
		select {
		case <-ctx.Done():
			return nil, false
		case <-ticker.C:
		}
	}
}
