package db

import "fmt"

// UpsertRateLimitCount atomically increments today's counter for the given IP
// and returns the new count. day is a UTC date string ("YYYY-MM-DD"); the
// (ip, day) primary key keeps the counter per-IP-per-UTC-day, and the single
// INSERT ... ON CONFLICT ... RETURNING statement keeps the read-modify-write
// atomic under concurrent requests. Inverse of the seed-insert idempotency
// pattern: seeds want at-most-one constant value; rate-limit wants monotonic
// increment where order matters and races would silently undercount.
func (db *DB) UpsertRateLimitCount(ip, day string) (int, error) {
	var count int
	err := db.conn.QueryRow(
		`INSERT INTO demo_rate_limits (ip, day, count) VALUES (?, ?, 1)
		 ON CONFLICT(ip, day) DO UPDATE SET count = count + 1
		 RETURNING count`,
		ip, day,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("upsert demo_rate_limits: %w", err)
	}
	return count, nil
}
