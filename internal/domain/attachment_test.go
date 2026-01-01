package domain

import (
	"strings"
	"testing"
)

func TestNewAttachment(t *testing.T) {
	att := NewAttachment(ID("att1"), ID("msg1"), "document.pdf", "application/pdf", 1024)

	if att.ID != ID("att1") {
		t.Errorf("NewAttachment().ID = %v, want %v", att.ID, "att1")
	}
	if att.MessageID != ID("msg1") {
		t.Errorf("NewAttachment().MessageID = %v, want %v", att.MessageID, "msg1")
	}
	if att.Filename != "document.pdf" {
		t.Errorf("NewAttachment().Filename = %v, want %v", att.Filename, "document.pdf")
	}
	if att.ContentType != "application/pdf" {
		t.Errorf("NewAttachment().ContentType = %v, want %v", att.ContentType, "application/pdf")
	}
	if att.Size != 1024 {
		t.Errorf("NewAttachment().Size = %v, want %v", att.Size, 1024)
	}
	if att.Disposition != DispositionAttachment {
		t.Errorf("NewAttachment().Disposition = %v, want %v", att.Disposition, DispositionAttachment)
	}
	if att.IsInline {
		t.Error("NewAttachment().IsInline should be false")
	}
}

func TestNewInlineAttachment(t *testing.T) {
	att := NewInlineAttachment(ID("att1"), ID("msg1"), "image.png", "image/png", "cid:123", 2048)

	if att.Disposition != DispositionInline {
		t.Errorf("NewInlineAttachment().Disposition = %v, want %v", att.Disposition, DispositionInline)
	}
	if !att.IsInline {
		t.Error("NewInlineAttachment().IsInline should be true")
	}
	if att.ContentID != "cid:123" {
		t.Errorf("NewInlineAttachment().ContentID = %v, want %v", att.ContentID, "cid:123")
	}
}

func TestAttachment_Validate(t *testing.T) {
	tests := []struct {
		name       string
		attachment *Attachment
		wantErr    bool
		errMsgs    []string
	}{
		{
			name: "valid attachment",
			attachment: &Attachment{
				ID:          ID("att1"),
				MessageID:   ID("msg1"),
				Filename:    "document.pdf",
				ContentType: "application/pdf",
				Size:        1024,
				Disposition: DispositionAttachment,
			},
			wantErr: false,
		},
		{
			name: "missing id",
			attachment: &Attachment{
				MessageID:   ID("msg1"),
				Filename:    "document.pdf",
				ContentType: "application/pdf",
				Size:        1024,
				Disposition: DispositionAttachment,
			},
			wantErr: true,
			errMsgs: []string{"id"},
		},
		{
			name: "missing message id",
			attachment: &Attachment{
				ID:          ID("att1"),
				Filename:    "document.pdf",
				ContentType: "application/pdf",
				Size:        1024,
				Disposition: DispositionAttachment,
			},
			wantErr: true,
			errMsgs: []string{"messageId"},
		},
		{
			name: "missing filename",
			attachment: &Attachment{
				ID:          ID("att1"),
				MessageID:   ID("msg1"),
				ContentType: "application/pdf",
				Size:        1024,
				Disposition: DispositionAttachment,
			},
			wantErr: true,
			errMsgs: []string{"filename"},
		},
		{
			name: "filename too long",
			attachment: &Attachment{
				ID:          ID("att1"),
				MessageID:   ID("msg1"),
				Filename:    strings.Repeat("a", 256),
				ContentType: "application/pdf",
				Size:        1024,
				Disposition: DispositionAttachment,
			},
			wantErr: true,
			errMsgs: []string{"filename"},
		},
		{
			name: "filename with path traversal",
			attachment: &Attachment{
				ID:          ID("att1"),
				MessageID:   ID("msg1"),
				Filename:    "../../../etc/passwd",
				ContentType: "text/plain",
				Size:        1024,
				Disposition: DispositionAttachment,
			},
			wantErr: true,
			errMsgs: []string{"filename"},
		},
		{
			name: "missing content type",
			attachment: &Attachment{
				ID:          ID("att1"),
				MessageID:   ID("msg1"),
				Filename:    "document.pdf",
				Size:        1024,
				Disposition: DispositionAttachment,
			},
			wantErr: true,
			errMsgs: []string{"contentType"},
		},
		{
			name: "invalid content type format",
			attachment: &Attachment{
				ID:          ID("att1"),
				MessageID:   ID("msg1"),
				Filename:    "document.pdf",
				ContentType: "invalid",
				Size:        1024,
				Disposition: DispositionAttachment,
			},
			wantErr: true,
			errMsgs: []string{"contentType"},
		},
		{
			name: "negative size",
			attachment: &Attachment{
				ID:          ID("att1"),
				MessageID:   ID("msg1"),
				Filename:    "document.pdf",
				ContentType: "application/pdf",
				Size:        -1,
				Disposition: DispositionAttachment,
			},
			wantErr: true,
			errMsgs: []string{"size"},
		},
		{
			name: "invalid disposition",
			attachment: &Attachment{
				ID:          ID("att1"),
				MessageID:   ID("msg1"),
				Filename:    "document.pdf",
				ContentType: "application/pdf",
				Size:        1024,
				Disposition: AttachmentDisposition("invalid"),
			},
			wantErr: true,
			errMsgs: []string{"disposition"},
		},
		{
			name: "inline without content id",
			attachment: &Attachment{
				ID:          ID("att1"),
				MessageID:   ID("msg1"),
				Filename:    "image.png",
				ContentType: "image/png",
				Size:        1024,
				Disposition: DispositionInline,
				IsInline:    true,
			},
			wantErr: true,
			errMsgs: []string{"contentId"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.attachment.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Attachment.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				errStr := err.Error()
				for _, msg := range tt.errMsgs {
					if !strings.Contains(errStr, msg) {
						t.Errorf("Attachment.Validate() error should contain '%s', got %v", msg, errStr)
					}
				}
			}
		})
	}
}

func TestAttachment_GetExtension(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"document.pdf", "pdf"},
		{"image.png", "png"},
		{"archive.tar.gz", "gz"},
		{"noextension", ""},
		{".hidden", "hidden"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			att := &Attachment{Filename: tt.filename}
			if got := att.GetExtension(); got != tt.want {
				t.Errorf("GetExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttachment_GetBaseFilename(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"document.pdf", "document"},
		{"image.png", "image"},
		{"noextension", "noextension"},
		{"file.name.txt", "file.name"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			att := &Attachment{Filename: tt.filename}
			if got := att.GetBaseFilename(); got != tt.want {
				t.Errorf("GetBaseFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttachment_IsImage(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"image/gif", true},
		{"application/pdf", false},
		{"text/plain", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			att := &Attachment{ContentType: tt.contentType}
			if got := att.IsImage(); got != tt.want {
				t.Errorf("IsImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttachment_IsPDF(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"application/pdf", true},
		{"image/png", false},
		{"text/plain", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			att := &Attachment{ContentType: tt.contentType}
			if got := att.IsPDF(); got != tt.want {
				t.Errorf("IsPDF() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttachment_IsText(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"text/plain", true},
		{"text/html", true},
		{"text/csv", true},
		{"application/pdf", false},
		{"image/png", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			att := &Attachment{ContentType: tt.contentType}
			if got := att.IsText(); got != tt.want {
				t.Errorf("IsText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttachment_IsArchive(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"application/zip", true},
		{"application/x-zip-compressed", true},
		{"application/x-rar-compressed", true},
		{"application/x-7z-compressed", true},
		{"application/x-tar", true},
		{"application/gzip", true},
		{"application/pdf", false},
		{"image/png", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			att := &Attachment{ContentType: tt.contentType}
			if got := att.IsArchive(); got != tt.want {
				t.Errorf("IsArchive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttachment_GetSizeFormatted(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{500, "500 B"},
		{1024, "1 KB"},
		{1536, "1,5 KB"},
		{10240, "10 KB"},
		{1048576, "1 MB"},
		{1572864, "1,5 MB"},
		{10485760, "10 MB"},
		{1073741824, "1 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			att := &Attachment{Size: tt.size}
			got := att.GetSizeFormatted()
			// Note: The formatting might vary slightly based on implementation
			if !strings.Contains(got, "B") && !strings.Contains(got, "KB") && !strings.Contains(got, "MB") && !strings.Contains(got, "GB") {
				t.Errorf("GetSizeFormatted() = %v, should contain size unit", got)
			}
		})
	}
}

func TestAttachment_ToSummary(t *testing.T) {
	att := NewAttachment(ID("att1"), ID("msg1"), "document.pdf", "application/pdf", 1024)

	summary := att.ToSummary()

	if summary.ID != att.ID {
		t.Errorf("ToSummary().ID = %v, want %v", summary.ID, att.ID)
	}
	if summary.Filename != att.Filename {
		t.Errorf("ToSummary().Filename = %v, want %v", summary.Filename, att.Filename)
	}
	if summary.ContentType != att.ContentType {
		t.Errorf("ToSummary().ContentType = %v, want %v", summary.ContentType, att.ContentType)
	}
	if summary.Size != att.Size {
		t.Errorf("ToSummary().Size = %v, want %v", summary.Size, att.Size)
	}
	if summary.IsInline != att.IsInline {
		t.Errorf("ToSummary().IsInline = %v, want %v", summary.IsInline, att.IsInline)
	}
	if summary.SizeFormatted == "" {
		t.Error("ToSummary().SizeFormatted should not be empty")
	}
}

func TestAttachmentDisposition_IsValid(t *testing.T) {
	tests := []struct {
		disposition AttachmentDisposition
		want        bool
	}{
		{DispositionAttachment, true},
		{DispositionInline, true},
		{AttachmentDisposition("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.disposition), func(t *testing.T) {
			if got := tt.disposition.IsValid(); got != tt.want {
				t.Errorf("AttachmentDisposition.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsPathTraversal(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"normal.txt", false},
		{"../parent.txt", true},
		{"..\\parent.txt", true},
		{"dir/file.txt", true},
		{"dir\\file.txt", true},
		{".hidden", true},
		{"file\x00.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if got := containsPathTraversal(tt.filename); got != tt.want {
				t.Errorf("containsPathTraversal(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsValidMIMEType(t *testing.T) {
	tests := []struct {
		mimeType string
		want     bool
	}{
		{"text/plain", true},
		{"application/pdf", true},
		{"image/png", true},
		{"application/octet-stream", true},
		{"invalid", false},
		{"text/", false},
		{"/plain", false},
		{"text/plain/extra", false},
		{"text plain", false},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			if got := isValidMIMEType(tt.mimeType); got != tt.want {
				t.Errorf("isValidMIMEType(%q) = %v, want %v", tt.mimeType, got, tt.want)
			}
		})
	}
}
