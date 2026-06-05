package persistence

import (
	"context"
	"time"
)

func (p *Postgres) LogAPIRequest(ctx context.Context, method, path string, statusCode int, userName string, durationMs int, clientIP string) error {
	if p == nil || p.Pool == nil {
		return nil
	}
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO contract_api_audit (method, path, status_code, user_name, duration_ms, client_ip)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, method, path, statusCode, userName, durationMs, clientIP)
	return err
}

func (p *Postgres) ListAPIAuditLogs(ctx context.Context, limit int) ([]map[string]any, int, error) {
	if p == nil || p.Pool == nil {
		return nil, 0, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var total int
	if err := p.Pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM contract_api_audit`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := p.Pool.Query(ctx, `
		SELECT method, path, status_code, user_name, duration_ms, logged_at, client_ip
		FROM contract_api_audit ORDER BY logged_at DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var method, path, user, ip string
		var status, dur int
		var at time.Time
		if err := rows.Scan(&method, &path, &status, &user, &dur, &at, &ip); err != nil {
			return nil, 0, err
		}
		out = append(out, map[string]any{
			"method": method, "path": path, "status": status,
			"user": user, "duration_ms": dur, "logged_at": at, "client_ip": ip,
		})
	}
	return out, total, rows.Err()
}

func (p *Postgres) APIMonitoringSummary(ctx context.Context) (map[string]any, error) {
	if p == nil || p.Pool == nil {
		return map[string]any{"requests_24h": 0, "errors_24h": 0, "avg_duration_ms": 0}, nil
	}
	var total24h, errors24h int
	var avgMs float64
	err := p.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE status_code >= 400)::int,
			COALESCE(AVG(duration_ms), 0)
		FROM contract_api_audit
		WHERE logged_at >= NOW() - INTERVAL '24 hours'
	`).Scan(&total24h, &errors24h, &avgMs)
	if err != nil {
		return nil, err
	}
	var contracts int
	_ = p.Pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM contracts`).Scan(&contracts)
	return map[string]any{
		"requests_24h":    total24h,
		"errors_24h":      errors24h,
		"avg_duration_ms": avgMs,
		"contracts":       contracts,
	}, nil
}

func (p *Postgres) APIMonitoringActivity(ctx context.Context, limit int) ([]map[string]any, error) {
	items, _, err := p.ListAPIAuditLogs(ctx, limit)
	return items, err
}

func (p *Postgres) MonitoringSummaryWithBus(ctx context.Context, busEnabled bool) (map[string]any, error) {
	summary, err := p.APIMonitoringSummary(ctx)
	if err != nil {
		return nil, err
	}
	summary["event_bus_enabled"] = busEnabled
	return summary, nil
}
