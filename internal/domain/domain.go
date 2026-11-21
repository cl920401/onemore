package domain

import "time"

type Channel struct {
	ID           int64
	Name         string
	BaseURL      string
	APIKeyMasked string
	Models       []string
	Enabled      bool
	CreatedAt    time.Time
}

type Member struct {
	ID                 int64
	Name               string
	Role               string
	Status             string
	EnterpriseLimit    int64
	EnterpriseUsed     int64
	EnterpriseReserved int64
	CreatedAt          time.Time
}

type Audit struct {
	RequestID        string
	CreatedAt        time.Time
	MemberID         int64
	MemberName       string
	ChannelID        int64
	QuotaSource      string
	Model            string
	Result           string
	ErrorCode        string
	HTTPStatus       int
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	LatencyMS        int64
}

type ConfigAudit struct {
	CreatedAt time.Time
	Actor     string
	Action    string
	Target    string
	Detail    string
}
