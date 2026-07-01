package authevent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// EnsureNextPartition creates a new monthly partition for auth_events if the
// next month's partition doesn't already exist. Call this periodically (e.g.
// daily) so writes never fail because a partition is missing.
func EnsureNextPartition(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	nextMonth := time.Now().AddDate(0, 1, 0)
	partName := partitionName(nextMonth)
	partStart := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	partEnd := partStart.AddDate(0, 1, 0)

	sql := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s PARTITION OF auth_events FOR VALUES FROM ('%s') TO ('%s')`,
		partName,
		partStart.Format("2006-01-02 15:04:05"),
		partEnd.Format("2006-01-02 15:04:05"),
	)
	if err := db.WithContext(ctx).Exec(sql).Error; err != nil {
		slog.Error("partition manager: failed to create next partition", "partition", partName, "err", err)
		return err
	}

	slog.Debug("partition manager: ensured next partition exists", "partition", partName)
	return nil
}

// StartPartitionManager runs a daily ticker that pre-creates the next month's
// partition. The goroutine exits when ctx is cancelled.
func StartPartitionManager(ctx context.Context, db *gorm.DB, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		if err := EnsureNextPartition(ctx, db); err != nil {
			slog.Warn("partition manager: initial ensure failed", "err", err)
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := EnsureNextPartition(ctx, db); err != nil {
					slog.Warn("partition manager: ensure failed", "err", err)
				}
			}
		}
	}()
}

// DropExpiredPartitions drops partitions that are entirely before the given
// cutoff. Returns the total number of partitions dropped.
func DropExpiredPartitions(ctx context.Context, db *gorm.DB, cutoff time.Time) (int64, error) {
	if db == nil {
		return 0, nil
	}
	var partitions []string
	err := db.WithContext(ctx).Raw(`
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
		  AND tablename LIKE 'auth_events_y%m%'
		ORDER BY tablename
	`).Scan(&partitions).Error
	if err != nil {
		return 0, err
	}

	var dropped int64
	for _, partName := range partitions {
		partEnd, err := partitionEndDate(partName)
		if err != nil {
			slog.Warn("partition manager: cannot parse partition name", "partition", partName, "err", err)
			continue
		}
		if partEnd.Before(cutoff) || partEnd.Equal(cutoff) {
			sql := fmt.Sprintf("DROP TABLE IF EXISTS %s", partName)
			if err := db.WithContext(ctx).Exec(sql).Error; err != nil {
				return dropped, err
			}
			dropped++
			slog.Info("partition manager: dropped expired partition", "partition", partName)
		}
	}
	return dropped, nil
}

func partitionName(t time.Time) string {
	return fmt.Sprintf("auth_events_y%04dm%02d", t.Year(), t.Month())
}

func partitionEndDate(name string) (time.Time, error) {
	var y, m int
	_, err := fmt.Sscanf(name, "auth_events_y%04dm%02d", &y, &m)
	if err != nil {
		return time.Time{}, err
	}
	end := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	return end, nil
}
