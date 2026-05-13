package parser

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BuildMessageOpts contains options for building a raw RFC 2822 message.
type BuildMessageOpts struct {
	From        string
	FromName    string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	TextBody    string
	HTMLBody    string
	Attachments []AttachmentInput
	Domain      string
}

// AttachmentInput represents an attachment to include in a built message.
type AttachmentInput struct {
	Filename    string
	ContentType string
	Data        []byte
}

// BuildRawMessage builds an RFC 2822 compliant MIME message.
// Returns two versions: forSent includes BCC header, forRelay omits it.
func BuildRawMessage(opts BuildMessageOpts) (forSent []byte, forRelay []byte) {
	messageID := fmt.Sprintf("<%s@%s>", uuid.New().String(), opts.Domain)
	date := time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")

	var common bytes.Buffer
	writeHeader(&common, "Date", date)
	writeHeader(&common, "From", formatAddress(opts.FromName, opts.From))
	writeHeader(&common, "To", formatAddressList(opts.To))
	if len(opts.Cc) > 0 {
		writeHeader(&common, "Cc", formatAddressList(opts.Cc))
	}
	writeHeader(&common, "Message-ID", messageID)
	writeEncodedHeader(&common, "Subject", opts.Subject)
	writeHeader(&common, "MIME-Version", "1.0")

	body := buildBody(opts)
	common.Write(body)

	// forRelay: no BCC header
	forRelay = common.Bytes()

	// forSent: with BCC header — insert after Cc/before Message-ID
	if len(opts.Bcc) > 0 {
		var sent bytes.Buffer
		writeHeader(&sent, "Date", date)
		writeHeader(&sent, "From", formatAddress(opts.FromName, opts.From))
		writeHeader(&sent, "To", formatAddressList(opts.To))
		if len(opts.Cc) > 0 {
			writeHeader(&sent, "Cc", formatAddressList(opts.Cc))
		}
		writeHeader(&sent, "Bcc", formatAddressList(opts.Bcc))
		writeHeader(&sent, "Message-ID", messageID)
		writeEncodedHeader(&sent, "Subject", opts.Subject)
		writeHeader(&sent, "MIME-Version", "1.0")
		sent.Write(body)
		forSent = sent.Bytes()
	} else {
		forSent = make([]byte, len(forRelay))
		copy(forSent, forRelay)
	}

	return forSent, forRelay
}

func buildBody(opts BuildMessageOpts) []byte {
	hasText := opts.TextBody != ""
	hasHTML := opts.HTMLBody != ""
	hasAttachments := len(opts.Attachments) > 0

	var buf bytes.Buffer

	switch {
	case hasAttachments:
		mixedBoundary := generateBoundary()
		buf.WriteString("Content-Type: multipart/mixed;\r\n\tboundary=\"" + mixedBoundary + "\"\r\n\r\n")
		buf.WriteString("--" + mixedBoundary + "\r\n")
		writeTextParts(&buf, opts.TextBody, opts.HTMLBody, hasText, hasHTML)
		for _, att := range opts.Attachments {
			buf.WriteString("\r\n--" + mixedBoundary + "\r\n")
			writeAttachmentPart(&buf, att)
		}
		buf.WriteString("\r\n--" + mixedBoundary + "--\r\n")

	case hasText && hasHTML:
		writeTextParts(&buf, opts.TextBody, opts.HTMLBody, true, true)

	case hasHTML:
		writeHeader(&buf, "Content-Type", "text/html; charset=UTF-8")
		writeHeader(&buf, "Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		writeQuotedPrintable(&buf, opts.HTMLBody)

	default:
		writeHeader(&buf, "Content-Type", "text/plain; charset=UTF-8")
		writeHeader(&buf, "Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		writeQuotedPrintable(&buf, opts.TextBody)
	}

	return buf.Bytes()
}

func writeTextParts(buf *bytes.Buffer, text, html string, hasText, hasHTML bool) {
	if hasText && hasHTML {
		altBoundary := generateBoundary()
		writeHeader(buf, "Content-Type", "multipart/alternative;\r\n\tboundary=\""+altBoundary+"\"")
		buf.WriteString("\r\n")
		buf.WriteString("--" + altBoundary + "\r\n")
		writeHeader(buf, "Content-Type", "text/plain; charset=UTF-8")
		writeHeader(buf, "Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		writeQuotedPrintable(buf, text)
		buf.WriteString("\r\n--" + altBoundary + "\r\n")
		writeHeader(buf, "Content-Type", "text/html; charset=UTF-8")
		writeHeader(buf, "Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		writeQuotedPrintable(buf, html)
		buf.WriteString("\r\n--" + altBoundary + "--\r\n")
	} else if hasText {
		writeHeader(buf, "Content-Type", "text/plain; charset=UTF-8")
		writeHeader(buf, "Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		writeQuotedPrintable(buf, text)
	} else if hasHTML {
		writeHeader(buf, "Content-Type", "text/html; charset=UTF-8")
		writeHeader(buf, "Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		writeQuotedPrintable(buf, html)
	}
}

func writeAttachmentPart(buf *bytes.Buffer, att AttachmentInput) {
	ct := att.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	encodedName := mime.QEncoding.Encode("utf-8", att.Filename)
	writeHeader(buf, "Content-Type", ct+"; name=\""+encodedName+"\"")
	writeHeader(buf, "Content-Transfer-Encoding", "base64")
	writeHeader(buf, "Content-Disposition", "attachment; filename=\""+encodedName+"\"")
	buf.WriteString("\r\n")
	writeBase64(buf, att.Data)
}

func writeHeader(buf *bytes.Buffer, name, value string) {
	buf.WriteString(name + ": " + value + "\r\n")
}

func writeEncodedHeader(buf *bytes.Buffer, name, value string) {
	if !isASCII(value) {
		value = mime.QEncoding.Encode("utf-8", value)
	}
	writeHeader(buf, name, value)
}

func formatAddress(name, addr string) string {
	if name == "" {
		return addr
	}
	encoded := name
	if !isASCII(name) {
		encoded = mime.QEncoding.Encode("utf-8", name)
	}
	return encoded + " <" + addr + ">"
}

func formatAddressList(addrs []string) string {
	if len(addrs) <= 3 {
		return strings.Join(addrs, ", ")
	}
	// Fold long recipient lists
	var buf bytes.Buffer
	for i, addr := range addrs {
		if i > 0 {
			buf.WriteString(",\r\n\t")
		}
		buf.WriteString(addr)
	}
	return buf.String()
}

func writeQuotedPrintable(buf *bytes.Buffer, text string) {
	w := quotedprintable.NewWriter(buf)
	w.Write([]byte(text))
	w.Close()
}

func writeBase64(buf *bytes.Buffer, data []byte) {
	encoded := base64.StdEncoding.EncodeToString(data)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		buf.WriteString(encoded[i:end])
		buf.WriteString("\r\n")
	}
}

func generateBoundary() string {
	return "----=_Part_" + uuid.New().String()
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}
