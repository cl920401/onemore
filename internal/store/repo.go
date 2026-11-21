package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"onemore/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// ChannelRecord carries the encrypted credential; handlers must not expose APIKeyEnc.
type ChannelRecord struct {
	domain.Channel
	APIKeyEnc []byte
}

func (r *Repo) CreateChannel(ctx context.Context, name, baseURL string, apiKey []byte, models []string, enabled bool) (int64, error) {
	modelsJSON, err := json.Marshal(models)
	if err != nil {
		return 0, fmt.Errorf("marshal models: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO channels (name, base_url, api_key, models, enabled) VALUES (?, ?, ?, ?, ?)`,
		name, baseURL, apiKey, modelsJSON, enabled)
	if err != nil {
		return 0, fmt.Errorf("insert channel: %w", err)
	}
	return res.LastInsertId()
}

func (r *Repo) ListChannels(ctx context.Context) ([]ChannelRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, base_url, api_key, models, enabled, created_at FROM channels ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	defer rows.Close()

	var out []ChannelRecord
	for rows.Next() {
		rec, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *Repo) GetChannel(ctx context.Context, id int64) (ChannelRecord, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, base_url, api_key, models, enabled, created_at FROM channels WHERE id = ?`, id)
	var rec ChannelRecord
	var modelsRaw []byte
	err := row.Scan(&rec.ID, &rec.Name, &rec.BaseURL, &rec.APIKeyEnc, &modelsRaw, &rec.Enabled, &rec.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ChannelRecord{}, ErrNotFound
	}
	if err != nil {
		return ChannelRecord{}, fmt.Errorf("get channel: %w", err)
	}
	if err := json.Unmarshal(modelsRaw, &rec.Models); err != nil {
		return ChannelRecord{}, fmt.Errorf("unmarshal channel models: %w", err)
	}
	return rec, nil
}

func (r *Repo) EnabledChannels(ctx context.Context) ([]ChannelRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, base_url, api_key, models, enabled, created_at FROM channels WHERE enabled = 1 ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("enabled channels: %w", err)
	}
	defer rows.Close()

	var out []ChannelRecord
	for rows.Next() {
		rec, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *Repo) SetChannelEnabled(ctx context.Context, id int64, enabled bool) error {
	res, err := r.db.ExecContext(ctx, `UPDATE channels SET enabled = ? WHERE id = ?`, enabled, id)
	if err != nil {
		return fmt.Errorf("set channel enabled: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanChannel(rows *sql.Rows) (ChannelRecord, error) {
	var rec ChannelRecord
	var modelsRaw []byte
	if err := rows.Scan(&rec.ID, &rec.Name, &rec.BaseURL, &rec.APIKeyEnc, &modelsRaw, &rec.Enabled, &rec.CreatedAt); err != nil {
		return rec, fmt.Errorf("scan channel: %w", err)
	}
	if err := json.Unmarshal(modelsRaw, &rec.Models); err != nil {
		return rec, fmt.Errorf("unmarshal channel models: %w", err)
	}
	return rec, nil
}

func (r *Repo) CreateMember(ctx context.Context, name, tokenHash string, limit int64) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO members (name, token_hash, enterprise_limit) VALUES (?, ?, ?)`,
		name, tokenHash, limit)
	if err != nil {
		return 0, fmt.Errorf("insert member: %w", err)
	}
	return res.LastInsertId()
}

func (r *Repo) ListMembers(ctx context.Context) ([]domain.Member, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, role, status, enterprise_limit, enterprise_used, enterprise_reserved, created_at
		 FROM members ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var out []domain.Member
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repo) MemberByTokenHash(ctx context.Context, hash string) (domain.Member, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, role, status, enterprise_limit, enterprise_used, enterprise_reserved, created_at
		 FROM members WHERE token_hash = ?`, hash)
	m, err := scanMember(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Member{}, ErrNotFound
	}
	return m, err
}

func (r *Repo) MemberByID(ctx context.Context, id int64) (domain.Member, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, role, status, enterprise_limit, enterprise_used, enterprise_reserved, created_at
		 FROM members WHERE id = ?`, id)
	m, err := scanMember(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Member{}, ErrNotFound
	}
	return m, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMember(row rowScanner) (domain.Member, error) {
	var m domain.Member
	err := row.Scan(&m.ID, &m.Name, &m.Role, &m.Status, &m.EnterpriseLimit,
		&m.EnterpriseUsed, &m.EnterpriseReserved, &m.CreatedAt)
	if err != nil {
		return m, err
	}
	return m, nil
}

func (r *Repo) SetMemberLimit(ctx context.Context, id, limit int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE members SET enterprise_limit = ? WHERE id = ?`, limit, id)
	if err != nil {
		return fmt.Errorf("set member limit: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) SetMemberStatus(ctx context.Context, id int64, status string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE members SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return fmt.Errorf("set member status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ReserveQuota atomically reserves amount against the enterprise ledger.
func (r *Repo) ReserveQuota(ctx context.Context, memberID, amount int64) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE members SET enterprise_reserved = enterprise_reserved + ?
		 WHERE id = ? AND status = 'active' AND enterprise_limit - enterprise_used - enterprise_reserved >= ?`,
		amount, memberID, amount)
	if err != nil {
		return false, fmt.Errorf("reserve quota: %w", err)
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (r *Repo) SettleQuota(ctx context.Context, memberID, reserved, actual int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE members SET enterprise_reserved = GREATEST(enterprise_reserved - ?, 0),
		                    enterprise_used = enterprise_used + ?
		 WHERE id = ?`, reserved, actual, memberID)
	if err != nil {
		return fmt.Errorf("settle quota: %w", err)
	}
	return nil
}

func (r *Repo) ReleaseQuota(ctx context.Context, memberID, amount int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE members SET enterprise_reserved = GREATEST(enterprise_reserved - ?, 0) WHERE id = ?`,
		amount, memberID)
	if err != nil {
		return fmt.Errorf("release quota: %w", err)
	}
	return nil
}

func (r *Repo) BlockedTerms(ctx context.Context) ([]string, error) {
	var raw []byte
	if err := r.db.QueryRowContext(ctx, `SELECT blocked_terms FROM policy WHERE id = 1`).Scan(&raw); err != nil {
		return nil, fmt.Errorf("get policy: %w", err)
	}
	var terms []string
	if err := json.Unmarshal(raw, &terms); err != nil {
		return nil, fmt.Errorf("unmarshal policy: %w", err)
	}
	return terms, nil
}

func (r *Repo) SetBlockedTerms(ctx context.Context, terms []string) error {
	raw, err := json.Marshal(terms)
	if err != nil {
		return fmt.Errorf("marshal policy: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `UPDATE policy SET blocked_terms = ? WHERE id = 1`, raw)
	if err != nil {
		return fmt.Errorf("set policy: %w", err)
	}
	return nil
}

func (r *Repo) InsertAudit(ctx context.Context, a domain.Audit) error {
	var channelID sql.NullInt64
	if a.ChannelID > 0 {
		channelID = sql.NullInt64{Int64: a.ChannelID, Valid: true}
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audits (request_id, member_id, member_name, channel_id, quota_source, model,
		                     result, error_code, http_status, prompt_tokens, completion_tokens, total_tokens, latency_ms)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.RequestID, a.MemberID, a.MemberName, channelID, a.QuotaSource, a.Model,
		a.Result, a.ErrorCode, a.HTTPStatus, a.PromptTokens, a.CompletionTokens, a.TotalTokens, a.LatencyMS)
	if err != nil {
		return fmt.Errorf("insert audit: %w", err)
	}
	return nil
}

func (r *Repo) ListAudits(ctx context.Context, limit int, requestID string) ([]domain.Audit, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT request_id, created_at, member_id, member_name, channel_id, quota_source, model,
	                 result, error_code, http_status, prompt_tokens, completion_tokens, total_tokens, latency_ms
	          FROM audits`
	args := make([]any, 0, 2)
	if requestID != "" {
		query += ` WHERE request_id = ?`
		args = append(args, requestID)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list audits: %w", err)
	}
	defer rows.Close()

	var out []domain.Audit
	for rows.Next() {
		var a domain.Audit
		var channelID sql.NullInt64
		if err := rows.Scan(&a.RequestID, &a.CreatedAt, &a.MemberID, &a.MemberName, &channelID,
			&a.QuotaSource, &a.Model, &a.Result, &a.ErrorCode, &a.HTTPStatus,
			&a.PromptTokens, &a.CompletionTokens, &a.TotalTokens, &a.LatencyMS); err != nil {
			return nil, fmt.Errorf("scan audit: %w", err)
		}
		a.ChannelID = channelID.Int64
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repo) InsertConfigAudit(ctx context.Context, actor, action, target, detail string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO config_audits (actor, action, target, detail) VALUES (?, ?, ?, ?)`,
		actor, action, target, detail)
	if err != nil {
		return fmt.Errorf("insert config audit: %w", err)
	}
	return nil
}
