package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestID_String(t *testing.T) {
	tests := []struct {
		name string
		id   ID
		want string
	}{
		{"empty id", ID(""), ""},
		{"simple id", ID("123"), "123"},
		{"uuid format", ID("550e8400-e29b-41d4-a716-446655440000"), "550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.String(); got != tt.want {
				t.Errorf("ID.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestID_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		id   ID
		want bool
	}{
		{"empty", ID(""), true},
		{"not empty", ID("123"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.IsEmpty(); got != tt.want {
				t.Errorf("ID.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestID_Value(t *testing.T) {
	id := ID("test-id")
	val, err := id.Value()
	if err != nil {
		t.Errorf("ID.Value() error = %v", err)
	}
	if val != "test-id" {
		t.Errorf("ID.Value() = %v, want %v", val, "test-id")
	}
}

func TestID_Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		want    ID
		wantErr bool
	}{
		{"scan string", "test-id", ID("test-id"), false},
		{"scan bytes", []byte("test-id"), ID("test-id"), false},
		{"scan nil", nil, ID(""), false},
		{"scan int", 123, ID(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id ID
			err := id.Scan(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ID.Scan() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && id != tt.want {
				t.Errorf("ID.Scan() = %v, want %v", id, tt.want)
			}
		})
	}
}

func TestEmailAddress_String(t *testing.T) {
	tests := []struct {
		name    string
		email   EmailAddress
		want    string
	}{
		{"address only", EmailAddress{Address: "test@example.com"}, "test@example.com"},
		{"with name", EmailAddress{Name: "John Doe", Address: "john@example.com"}, "John Doe <john@example.com>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.email.String(); got != tt.want {
				t.Errorf("EmailAddress.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmailAddress_IsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		email EmailAddress
		want  bool
	}{
		{"empty", EmailAddress{}, true},
		{"with address", EmailAddress{Address: "test@example.com"}, false},
		{"name only", EmailAddress{Name: "John"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.email.IsEmpty(); got != tt.want {
				t.Errorf("EmailAddress.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimestamp_MarshalJSON(t *testing.T) {
	ts := Timestamp{Time: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)}
	data, err := json.Marshal(ts)
	if err != nil {
		t.Errorf("Timestamp.MarshalJSON() error = %v", err)
	}
	expected := `"2025-01-15T10:30:00Z"`
	if string(data) != expected {
		t.Errorf("Timestamp.MarshalJSON() = %v, want %v", string(data), expected)
	}
}

func TestTimestamp_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{"valid RFC3339", `"2025-01-15T10:30:00Z"`, false},
		{"invalid format", `"2025/01/15"`, true},
		{"not a string", `12345`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts Timestamp
			err := json.Unmarshal([]byte(tt.data), &ts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Timestamp.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNow(t *testing.T) {
	before := time.Now().UTC()
	ts := Now()
	after := time.Now().UTC()

	if ts.Time.Before(before) || ts.Time.After(after) {
		t.Errorf("Now() returned time outside expected range")
	}
}

func TestUserRole_IsValid(t *testing.T) {
	tests := []struct {
		role UserRole
		want bool
	}{
		{RoleAdmin, true},
		{RoleUser, true},
		{RoleViewer, true},
		{UserRole("invalid"), false},
		{UserRole(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.want {
				t.Errorf("UserRole.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserStatus_IsValid(t *testing.T) {
	tests := []struct {
		status UserStatus
		want   bool
	}{
		{StatusActive, true},
		{StatusInactive, true},
		{StatusPending, true},
		{UserStatus("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("UserStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageStatus_IsValid(t *testing.T) {
	tests := []struct {
		status MessageStatus
		want   bool
	}{
		{MessageUnread, true},
		{MessageRead, true},
		{MessageStatus("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("MessageStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWebhookEvent_IsValid(t *testing.T) {
	tests := []struct {
		event WebhookEvent
		want  bool
	}{
		{WebhookEventMessageReceived, true},
		{WebhookEventMessageDeleted, true},
		{WebhookEventMailboxCreated, true},
		{WebhookEventMailboxDeleted, true},
		{WebhookEvent("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.event), func(t *testing.T) {
			if got := tt.event.IsValid(); got != tt.want {
				t.Errorf("WebhookEvent.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWebhookStatus_IsValid(t *testing.T) {
	tests := []struct {
		status WebhookStatus
		want   bool
	}{
		{WebhookStatusActive, true},
		{WebhookStatusInactive, true},
		{WebhookStatusFailed, true},
		{WebhookStatus("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("WebhookStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDatabaseDriver_IsValid(t *testing.T) {
	tests := []struct {
		driver DatabaseDriver
		want   bool
	}{
		{DatabaseDriverSQLite, true},
		{DatabaseDriverPostgres, true},
		{DatabaseDriverMySQL, true},
		{DatabaseDriverMongoDB, true},
		{DatabaseDriver("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.driver), func(t *testing.T) {
			if got := tt.driver.IsValid(); got != tt.want {
				t.Errorf("DatabaseDriver.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPagination_Offset(t *testing.T) {
	tests := []struct {
		name    string
		page    int
		perPage int
		want    int
	}{
		{"page 1", 1, 10, 0},
		{"page 2", 2, 10, 10},
		{"page 3", 3, 25, 50},
		{"page 0", 0, 10, 0},
		{"negative page", -1, 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pagination{Page: tt.page, PerPage: tt.perPage}
			if got := p.Offset(); got != tt.want {
				t.Errorf("Pagination.Offset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPagination_TotalPages(t *testing.T) {
	tests := []struct {
		name    string
		total   int64
		perPage int
		want    int
	}{
		{"exact pages", 100, 10, 10},
		{"partial page", 105, 10, 11},
		{"single page", 5, 10, 1},
		{"zero items", 0, 10, 0},
		{"zero per page", 100, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pagination{Total: tt.total, PerPage: tt.perPage}
			if got := p.TotalPages(); got != tt.want {
				t.Errorf("Pagination.TotalPages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPagination_HasNext(t *testing.T) {
	tests := []struct {
		name    string
		page    int
		perPage int
		total   int64
		want    bool
	}{
		{"has next", 1, 10, 100, true},
		{"last page", 10, 10, 100, false},
		{"single page", 1, 10, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pagination{Page: tt.page, PerPage: tt.perPage, Total: tt.total}
			if got := p.HasNext(); got != tt.want {
				t.Errorf("Pagination.HasNext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPagination_HasPrev(t *testing.T) {
	tests := []struct {
		name string
		page int
		want bool
	}{
		{"first page", 1, false},
		{"second page", 2, true},
		{"later page", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pagination{Page: tt.page}
			if got := p.HasPrev(); got != tt.want {
				t.Errorf("Pagination.HasPrev() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortOrder_IsValid(t *testing.T) {
	tests := []struct {
		order SortOrder
		want  bool
	}{
		{SortAsc, true},
		{SortDesc, true},
		{SortOrder("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.order), func(t *testing.T) {
			if got := tt.order.IsValid(); got != tt.want {
				t.Errorf("SortOrder.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
