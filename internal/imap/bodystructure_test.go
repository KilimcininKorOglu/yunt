package imap

import (
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"

	"yunt/internal/domain"
)

func TestBodyStructureBuilder_ExtractContentType(t *testing.T) {
	builder := &BodyStructureBuilder{}

	tests := []struct {
		name    string
		rawBody []byte
		want    string
	}{
		{
			name:    "simple text/plain",
			rawBody: []byte("Content-Type: text/plain\r\n\r\nBody"),
			want:    "text/plain",
		},
		{
			name:    "with charset",
			rawBody: []byte("Content-Type: text/html; charset=utf-8\r\n\r\nBody"),
			want:    "text/html",
		},
		{
			name:    "multipart",
			rawBody: []byte("Content-Type: multipart/mixed; boundary=\"abc\"\r\n\r\nBody"),
			want:    "multipart/mixed",
		},
		{
			name:    "no content type",
			rawBody: []byte("Subject: Test\r\n\r\nBody"),
			want:    "text/plain",
		},
		{
			name:    "case insensitive",
			rawBody: []byte("CONTENT-TYPE: TEXT/HTML\r\n\r\nBody"),
			want:    "TEXT/HTML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.extractContentType(tt.rawBody)
			if got != tt.want {
				t.Errorf("extractContentType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBodyStructureBuilder_ExtractBoundary(t *testing.T) {
	builder := &BodyStructureBuilder{}

	tests := []struct {
		name    string
		rawBody []byte
		want    string
	}{
		{
			name:    "simple boundary",
			rawBody: []byte("Content-Type: multipart/mixed; boundary=simple\r\n\r\n"),
			want:    "simple",
		},
		{
			name:    "quoted boundary",
			rawBody: []byte(`Content-Type: multipart/mixed; boundary="quoted-boundary"` + "\r\n\r\n"),
			want:    "quoted-boundary",
		},
		{
			name:    "boundary with special chars",
			rawBody: []byte(`Content-Type: multipart/mixed; boundary="=_Part_123"` + "\r\n\r\n"),
			want:    "=_Part_123",
		},
		{
			name:    "no boundary",
			rawBody: []byte("Content-Type: text/plain\r\n\r\n"),
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.extractBoundary(tt.rawBody)
			if got != tt.want {
				t.Errorf("extractBoundary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBodyStructureBuilder_SplitByBoundary(t *testing.T) {
	builder := &BodyStructureBuilder{}

	rawBody := []byte(`Content-Type: multipart/mixed; boundary="myboundary"

--myboundary
Content-Type: text/plain

Part 1
--myboundary
Content-Type: text/html

Part 2
--myboundary--`)

	parts := builder.splitByBoundary(rawBody, "myboundary")

	if len(parts) != 2 {
		t.Fatalf("Expected 2 parts, got %d", len(parts))
	}

	if !strings.Contains(string(parts[0]), "Part 1") {
		t.Error("Part 1 content not found")
	}

	if !strings.Contains(string(parts[1]), "Part 2") {
		t.Error("Part 2 content not found")
	}
}

func TestBodyStructureBuilder_ExtractHeaderValue(t *testing.T) {
	builder := &BodyStructureBuilder{}

	rawBody := []byte("Content-Type: text/plain\r\nContent-ID: <abc123>\r\nContent-Transfer-Encoding: base64\r\n\r\nBody")

	tests := []struct {
		header string
		want   string
	}{
		{"Content-ID", "<abc123>"},
		{"Content-Transfer-Encoding", "base64"},
		{"X-Missing", ""},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			got := builder.extractHeaderValue(rawBody, tt.header)
			if got != tt.want {
				t.Errorf("extractHeaderValue(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

func TestBodyStructureBuilder_ExtractContentTypeParams(t *testing.T) {
	builder := &BodyStructureBuilder{}

	rawBody := []byte(`Content-Type: text/plain; charset="utf-8"; format=flowed` + "\r\n\r\nBody")

	params := builder.extractContentTypeParams(rawBody)

	if params["charset"] != "utf-8" {
		t.Errorf("charset = %q, want %q", params["charset"], "utf-8")
	}

	if params["format"] != "flowed" {
		t.Errorf("format = %q, want %q", params["format"], "flowed")
	}
}

func TestBodyStructureBuilder_ExtractDisposition(t *testing.T) {
	builder := &BodyStructureBuilder{}

	tests := []struct {
		name    string
		rawBody []byte
		wantVal string
	}{
		{
			name:    "attachment disposition",
			rawBody: []byte(`Content-Disposition: attachment; filename="test.pdf"` + "\r\n\r\n"),
			wantVal: "attachment",
		},
		{
			name:    "inline disposition",
			rawBody: []byte("Content-Disposition: inline\r\n\r\n"),
			wantVal: "inline",
		},
		{
			name:    "no disposition",
			rawBody: []byte("Content-Type: text/plain\r\n\r\n"),
			wantVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			disp := builder.extractDisposition(tt.rawBody)
			if tt.wantVal == "" {
				if disp != nil {
					t.Errorf("Expected nil disposition, got %v", disp)
				}
			} else {
				if disp == nil {
					t.Errorf("Expected disposition, got nil")
				} else if disp.Value != tt.wantVal {
					t.Errorf("disposition.Value = %q, want %q", disp.Value, tt.wantVal)
				}
			}
		})
	}
}

func TestBodyStructureBuilder_CountLines(t *testing.T) {
	builder := &BodyStructureBuilder{}

	tests := []struct {
		data []byte
		want int
	}{
		{[]byte(""), 0},
		{[]byte("single line"), 1},
		{[]byte("line1\nline2"), 2},
		{[]byte("line1\nline2\nline3"), 3},
		{[]byte("line1\r\nline2"), 2},
	}

	for _, tt := range tests {
		got := builder.countLines(tt.data)
		if got != tt.want {
			t.Errorf("countLines(%q) = %d, want %d", string(tt.data), got, tt.want)
		}
	}
}

func TestBodyStructureBuilder_ExtractBody(t *testing.T) {
	builder := &BodyStructureBuilder{}

	tests := []struct {
		name    string
		rawBody []byte
		want    string
	}{
		{
			name:    "CRLF separator",
			rawBody: []byte("Header: value\r\n\r\nBody content"),
			want:    "Body content",
		},
		{
			name:    "LF separator",
			rawBody: []byte("Header: value\n\nBody content"),
			want:    "Body content",
		},
		{
			name:    "no body",
			rawBody: []byte("Header: value"),
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.extractBody(tt.rawBody)
			if string(got) != tt.want {
				t.Errorf("extractBody() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestBodyStructureBuilder_BuildSinglePartFromMessage(t *testing.T) {
	builder := NewBodyStructureBuilder()

	tests := []struct {
		name     string
		msg      *domain.Message
		wantType string
		wantSub  string
	}{
		{
			name: "plain text message",
			msg: &domain.Message{
				TextBody: "Hello, world!",
			},
			wantType: "TEXT",
			wantSub:  "PLAIN",
		},
		{
			name: "HTML message",
			msg: &domain.Message{
				HTMLBody: "<p>Hello, world!</p>",
			},
			wantType: "TEXT",
			wantSub:  "HTML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs, err := builder.buildSinglePartFromMessage(tt.msg, false)
			if err != nil {
				t.Fatalf("buildSinglePartFromMessage() error = %v", err)
			}

			singlePart, ok := bs.(*imap.BodyStructureSinglePart)
			if !ok {
				t.Fatalf("Expected BodyStructureSinglePart, got %T", bs)
			}

			if singlePart.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", singlePart.Type, tt.wantType)
			}

			if singlePart.Subtype != tt.wantSub {
				t.Errorf("Subtype = %q, want %q", singlePart.Subtype, tt.wantSub)
			}
		})
	}
}

func TestBodyStructureBuilder_AttachmentToBodyStructure(t *testing.T) {
	builder := NewBodyStructureBuilder()

	att := &domain.Attachment{
		ID:          domain.ID("att-1"),
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		Size:        12345,
		IsInline:    false,
	}

	bs := builder.attachmentToBodyStructure(att, true)

	singlePart, ok := bs.(*imap.BodyStructureSinglePart)
	if !ok {
		t.Fatalf("Expected BodyStructureSinglePart, got %T", bs)
	}

	if singlePart.Type != "APPLICATION" {
		t.Errorf("Type = %q, want APPLICATION", singlePart.Type)
	}

	if singlePart.Subtype != "PDF" {
		t.Errorf("Subtype = %q, want PDF", singlePart.Subtype)
	}

	if singlePart.Size != 12345 {
		t.Errorf("Size = %d, want %d", singlePart.Size, 12345)
	}

	if singlePart.Extended == nil {
		t.Error("Expected Extended to be set")
	} else if singlePart.Extended.Disposition == nil {
		t.Error("Expected Disposition to be set")
	} else if singlePart.Extended.Disposition.Value != "attachment" {
		t.Errorf("Disposition = %q, want attachment", singlePart.Extended.Disposition.Value)
	}
}

func TestBodyStructureBuilder_InlineAttachment(t *testing.T) {
	builder := NewBodyStructureBuilder()

	att := &domain.Attachment{
		ID:          domain.ID("att-2"),
		Filename:    "image.png",
		ContentType: "image/png",
		ContentID:   "cid123",
		Size:        5000,
		IsInline:    true,
	}

	bs := builder.attachmentToBodyStructure(att, true)

	singlePart, ok := bs.(*imap.BodyStructureSinglePart)
	if !ok {
		t.Fatalf("Expected BodyStructureSinglePart, got %T", bs)
	}

	if singlePart.Type != "IMAGE" {
		t.Errorf("Type = %q, want IMAGE", singlePart.Type)
	}

	if singlePart.ID != "cid123" {
		t.Errorf("ID = %q, want cid123", singlePart.ID)
	}

	if singlePart.Extended.Disposition.Value != "inline" {
		t.Errorf("Disposition = %q, want inline", singlePart.Extended.Disposition.Value)
	}
}
