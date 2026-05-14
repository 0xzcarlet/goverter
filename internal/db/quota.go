package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func reserveDailySlot(ctx context.Context, tx pgx.Tx, userID string, quotaDate time.Time, limit int) error {
	quotaDateArg := quotaDateValue(quotaDate)
	_, err := tx.Exec(ctx, `
		insert into app.daily_conversion_usage (user_id, quota_date, reserved_count, completed_count)
		values ($1, $2, 0, 0)
		on conflict (user_id, quota_date) do nothing
	`, userID, quotaDateArg)
	if err != nil {
		return err
	}

	var reservedCount int
	var completedCount int
	err = tx.QueryRow(ctx, `
		select reserved_count, completed_count
		from app.daily_conversion_usage
		where user_id = $1 and quota_date = $2
		for update
	`, userID, quotaDateArg).Scan(&reservedCount, &completedCount)
	if err != nil {
		return err
	}

	if reservedCount+completedCount >= limit {
		return ErrDailyLimitReached
	}

	_, err = tx.Exec(ctx, `
		update app.daily_conversion_usage
		set reserved_count = reserved_count + 1,
			updated_at = now()
		where user_id = $1 and quota_date = $2
	`, userID, quotaDateArg)
	return err
}

func adjustUsageCounts(ctx context.Context, tx pgx.Tx, userID string, quotaDate time.Time, reservedDelta, completedDelta int) error {
	quotaDateArg := quotaDateValue(quotaDate)
	commandTag, err := tx.Exec(ctx, `
		update app.daily_conversion_usage
		set reserved_count = reserved_count + $3,
			completed_count = completed_count + $4,
			updated_at = now()
		where user_id = $1 and quota_date = $2
	`, userID, quotaDateArg, reservedDelta, completedDelta)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf("daily usage row missing for user %s on %s", userID, quotaDate.Format("2006-01-02"))
	}
	return nil
}

func quotaDateValue(value time.Time) string {
	return value.Format("2006-01-02")
}
