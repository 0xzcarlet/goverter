package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) GuestDailyUsageSummary(ctx context.Context, guestToken string, quotaDate time.Time) (GuestDailyUsage, error) {
	const query = `
		select guest_token, quota_date, reserved_count, completed_count
		from app.guest_daily_conversion_usage
		where guest_token = $1 and quota_date = $2
	`

	var usage GuestDailyUsage
	err := r.pool.QueryRow(ctx, query, guestToken, quotaDateValue(quotaDate)).Scan(
		&usage.GuestToken,
		&usage.QuotaDate,
		&usage.ReservedCount,
		&usage.CompletedCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GuestDailyUsage{GuestToken: guestToken, QuotaDate: quotaDate}, nil
		}
		return GuestDailyUsage{}, err
	}
	return usage, nil
}

func (r *Repository) ReserveGuestDailySlot(ctx context.Context, guestToken string, quotaDate time.Time, limit int) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := reserveGuestDailySlot(ctx, tx, guestToken, quotaDate, limit); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) CompleteGuestDailySlot(ctx context.Context, guestToken string, quotaDate time.Time) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := adjustGuestUsageCounts(ctx, tx, guestToken, quotaDate, -1, 1); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) RefundGuestDailySlot(ctx context.Context, guestToken string, quotaDate time.Time) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := adjustGuestUsageCounts(ctx, tx, guestToken, quotaDate, -1, 0); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func reserveGuestDailySlot(ctx context.Context, tx pgx.Tx, guestToken string, quotaDate time.Time, limit int) error {
	quotaDateArg := quotaDateValue(quotaDate)
	_, err := tx.Exec(ctx, `
		insert into app.guest_daily_conversion_usage (guest_token, quota_date, reserved_count, completed_count)
		values ($1, $2, 0, 0)
		on conflict (guest_token, quota_date) do nothing
	`, guestToken, quotaDateArg)
	if err != nil {
		return err
	}

	var reservedCount int
	var completedCount int
	err = tx.QueryRow(ctx, `
		select reserved_count, completed_count
		from app.guest_daily_conversion_usage
		where guest_token = $1 and quota_date = $2
		for update
	`, guestToken, quotaDateArg).Scan(&reservedCount, &completedCount)
	if err != nil {
		return err
	}

	if reservedCount+completedCount >= limit {
		return ErrDailyLimitReached
	}

	_, err = tx.Exec(ctx, `
		update app.guest_daily_conversion_usage
		set reserved_count = reserved_count + 1,
			updated_at = now()
		where guest_token = $1 and quota_date = $2
	`, guestToken, quotaDateArg)
	return err
}

func adjustGuestUsageCounts(ctx context.Context, tx pgx.Tx, guestToken string, quotaDate time.Time, reservedDelta, completedDelta int) error {
	commandTag, err := tx.Exec(ctx, `
		update app.guest_daily_conversion_usage
		set reserved_count = reserved_count + $3,
			completed_count = completed_count + $4,
			updated_at = now()
		where guest_token = $1 and quota_date = $2
	`, guestToken, quotaDateValue(quotaDate), reservedDelta, completedDelta)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("guest daily usage row missing")
	}
	return nil
}
