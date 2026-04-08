package postgres

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func formatInterval(d time.Duration) string {
	return fmt.Sprintf("%d microseconds", d.Microseconds())
}

func intervalToDuration(iv pgtype.Interval) (time.Duration, error) {
	if !iv.Valid {
		return 0, nil
	}
	if iv.Months != 0 {
		return 0, fmt.Errorf("postgres interval with months is not supported: %+v", iv)
	}

	d := time.Duration(iv.Days) * 24 * time.Hour
	d += time.Duration(iv.Microseconds) * time.Microsecond
	return d, nil
}
