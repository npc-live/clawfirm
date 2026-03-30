package gateway

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// extractDocumentText attempts to extract plain text from a document file.
// Returns empty string if the format is not supported or parsing fails.
func extractDocumentText(data []byte, fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".docx":
		return extractDocx(data)
	case ".txt", ".md", ".csv", ".json", ".xml", ".yaml", ".yml", ".html", ".htm":
		if utf8.Valid(data) {
			return string(data)
		}
	}
	return ""
}

// extractDocx extracts plain text from a .docx file (ZIP+XML).
// It reads word/document.xml and strips all XML tags.
func extractDocx(data []byte) string {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return ""
	}

	for _, f := range r.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return ""
		}
		text := extractXMLText(rc)
		rc.Close()
		return text
	}
	return ""
}

// extractXMLText reads an XML stream and returns all character data concatenated,
// inserting newlines between paragraph-level elements (w:p).
func extractXMLText(r io.Reader) string {
	var sb strings.Builder
	dec := xml.NewDecoder(r)
	inParagraph := false

	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			// w:p = paragraph boundary
			if t.Name.Local == "p" && t.Name.Space != "" {
				inParagraph = true
			}
		case xml.EndElement:
			if t.Name.Local == "p" && inParagraph {
				sb.WriteByte('\n')
				inParagraph = false
			}
		case xml.CharData:
			s := strings.TrimSpace(string(t))
			if s != "" {
				if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "\n") {
					sb.WriteByte(' ')
				}
				sb.WriteString(s)
			}
		}
	}
	return strings.TrimSpace(sb.String())
}
