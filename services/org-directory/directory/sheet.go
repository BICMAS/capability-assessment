package directory

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode/utf8"
)

const maxSheetBytes = 8 << 20

func classifyOrganogramFile(fileName string, body []byte) (string, string, error) {
	if len(body) == 0 || len(body) > MaxOrganogramBytes {
		if len(body) > MaxOrganogramBytes {
			return "", "", ErrTooLarge
		}
		return "", "", ErrInvalid
	}
	extension := strings.ToLower(path.Ext(fileName))
	switch {
	case extension == ".png" && hasPrefix(body, []byte{0x89, 'P', 'N', 'G'}):
		return "image/png", "png", nil
	case (extension == ".jpg" || extension == ".jpeg") && hasPrefix(body, []byte{0xff, 0xd8, 0xff}):
		return "image/jpeg", "jpeg", nil
	case extension == ".pdf" && hasPrefix(body, []byte("%PDF")):
		return "application/pdf", "pdf", nil
	case extension == ".xls" && hasPrefix(body, []byte{0xd0, 0xcf, 0x11, 0xe0}):
		return "application/vnd.ms-excel", "xls", nil
	case extension == ".xlsx" && hasPrefix(body, []byte{'P', 'K'}) && validXLSX(body):
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx", nil
	default:
		return "", "", ErrInvalid
	}
}

func hasPrefix(body, prefix []byte) bool {
	return len(body) >= len(prefix) && bytes.Equal(body[:len(prefix)], prefix)
}

func validXLSX(body []byte) bool {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return false
	}
	for _, file := range reader.File {
		if file.Name == "xl/workbook.xml" {
			return true
		}
	}
	return false
}

func TemplateXLSX() ([]byte, error) {
	files := []struct {
		name string
		body string
	}{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`},
		{"xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Organogram" sheetId="1" r:id="rId1"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`},
		{"xl/worksheets/sheet1.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>title</t></is></c><c r="B1" t="inlineStr"><is><t>reports_to</t></is></c><c r="C1" t="inlineStr"><is><t>department</t></is></c></row></sheetData></worksheet>`},
	}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, file := range files {
		entry, err := writer.Create(file.name)
		if err != nil {
			return nil, err
		}
		if _, err := entry.Write([]byte(file.body)); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func readXLSX(body []byte) ([][]string, error) {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}
	var shared []string
	var sheet []byte
	var read int64
	for _, file := range reader.File {
		if file.Name != "xl/sharedStrings.xml" && !strings.HasPrefix(file.Name, "xl/worksheets/") {
			continue
		}
		if file.UncompressedSize64 > maxSheetBytes {
			return nil, fmt.Errorf("spreadsheet is too large")
		}
		contents, err := readZipFile(file, maxSheetBytes-read)
		if err != nil {
			return nil, err
		}
		read += int64(len(contents))
		if file.Name == "xl/sharedStrings.xml" {
			shared, err = sharedStrings(contents)
			if err != nil {
				return nil, err
			}
		}
		if strings.HasPrefix(file.Name, "xl/worksheets/") && strings.HasSuffix(file.Name, ".xml") && sheet == nil {
			sheet = contents
		}
	}
	if sheet == nil {
		return nil, fmt.Errorf("spreadsheet has no worksheet")
	}
	return worksheetRows(sheet, shared)
}

func readZipFile(file *zip.File, limit int64) ([]byte, error) {
	if limit < 0 {
		return nil, fmt.Errorf("spreadsheet is too large")
	}
	entry, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer entry.Close()
	return io.ReadAll(io.LimitReader(entry, limit+1))
}

type xlsxSST struct {
	Items []xlsxSI `xml:"si"`
}

type xlsxSI struct {
	Text []string `xml:"t"`
	Rich []xlsxR  `xml:"r"`
}

type xlsxR struct {
	Text []string `xml:"t"`
}

func sharedStrings(body []byte) ([]string, error) {
	var table xlsxSST
	if err := xml.Unmarshal(body, &table); err != nil {
		return nil, err
	}
	values := make([]string, 0, len(table.Items))
	for _, item := range table.Items {
		text := strings.Join(item.Text, "")
		for _, run := range item.Rich {
			text += strings.Join(run.Text, "")
		}
		values = append(values, text)
	}
	return values, nil
}

type xlsxWorksheet struct {
	Rows []xlsxRow `xml:"sheetData>row"`
}

type xlsxRow struct {
	Number string     `xml:"r,attr"`
	Cells  []xlsxCell `xml:"c"`
}

type xlsxCell struct {
	Ref    string     `xml:"r,attr"`
	Type   string     `xml:"t,attr"`
	Value  string     `xml:"v"`
	Inline xlsxInline `xml:"is"`
}

type xlsxInline struct {
	Text []string `xml:"t"`
}

func worksheetRows(body []byte, shared []string) ([][]string, error) {
	var sheet xlsxWorksheet
	if err := xml.Unmarshal(body, &sheet); err != nil {
		return nil, err
	}
	rows := make([][]string, 0, len(sheet.Rows))
	for _, row := range sheet.Rows {
		width := 0
		for _, cell := range row.Cells {
			index := columnIndex(cell.Ref)
			if index+1 > width {
				width = index + 1
			}
		}
		values := make([]string, width)
		for _, cell := range row.Cells {
			index := columnIndex(cell.Ref)
			if index < 0 || index >= len(values) {
				continue
			}
			values[index] = cellText(cell, shared)
		}
		rows = append(rows, values)
	}
	return rows, nil
}

func cellText(cell xlsxCell, shared []string) string {
	if cell.Type == "inlineStr" {
		return strings.Join(cell.Inline.Text, "")
	}
	if cell.Type == "s" {
		index := 0
		for _, digit := range strings.TrimSpace(cell.Value) {
			if digit < '0' || digit > '9' {
				return ""
			}
			index = index*10 + int(digit-'0')
		}
		if index >= 0 && index < len(shared) && cell.Value != "" {
			return shared[index]
		}
		return ""
	}
	return cell.Value
}

func columnIndex(ref string) int {
	if ref == "" {
		return -1
	}
	index := 0
	seen := false
	for _, char := range ref {
		if char >= '0' && char <= '9' {
			break
		}
		upper := char
		if upper >= 'a' && upper <= 'z' {
			upper = upper - 'a' + 'A'
		}
		if upper < 'A' || upper > 'Z' {
			return -1
		}
		seen = true
		index = index*26 + int(upper-'A'+1)
	}
	if !seen {
		return -1
	}
	return index - 1
}

func xmlText(value string) string {
	var buffer strings.Builder
	for _, char := range value {
		switch char {
		case '&':
			buffer.WriteString("&amp;")
		case '<':
			buffer.WriteString("&lt;")
		case '>':
			buffer.WriteString("&gt;")
		default:
			if utf8.ValidRune(char) {
				buffer.WriteRune(char)
			}
		}
	}
	return buffer.String()
}
