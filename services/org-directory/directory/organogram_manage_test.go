package directory

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bicmas-academy/learn/services/token"
)

func TestOrganogramCanBeBuiltSavedReloadedAndGuarded(t *testing.T) {
	router := NewRouter(nil, []byte(secret), seeded(), &recordingPublisher{})
	hr := token.Claims{PersonID: adaID, OrganisationID: organisationID, Roles: []string{"hr"}}
	organogram := createChart(t, router, hr, "Sales chart")
	if organogram["status"] != OrganogramDraft {
		t.Fatalf("status = %#v", organogram["status"])
	}
	director := createChartPosition(t, router, hr, organogram["id"].(string), `{"title":"Managing Director"}`)
	manager := createChartPosition(t, router, hr, organogram["id"].(string), `{"title":"Sales Manager","reportsToPositionId":"`+director["id"].(string)+`"}`)
	lead := createChartPosition(t, router, hr, organogram["id"].(string), `{"title":"Team Lead","departmentId":"c4e8a1b2-7d63-4f90-8a15-2b6c9d0e3f71","reportsToPositionId":"`+manager["id"].(string)+`"}`)

	moved := patchPosition(t, router, hr, organogram["id"].(string), lead["id"].(string), `{"title":"Team Lead","departmentId":"c4e8a1b2-7d63-4f90-8a15-2b6c9d0e3f71","personId":"","reportsToPositionId":"`+director["id"].(string)+`"}`)
	if moved["reportsToPositionId"] != director["id"] {
		t.Fatalf("moved = %#v", moved)
	}
	cycle := httptest.NewRequest(http.MethodPatch, "/organograms/"+organogram["id"].(string)+"/positions/"+director["id"].(string), strings.NewReader(`{"title":"Managing Director","reportsToPositionId":"`+lead["id"].(string)+`"}`))
	cycle.Header.Set("Authorization", "Bearer "+sign(t, hr))
	cycleResponse := httptest.NewRecorder()
	router.ServeHTTP(cycleResponse, cycle)
	if cycleResponse.Code != http.StatusBadRequest || !strings.Contains(cycleResponse.Body.String(), "cycle") {
		t.Fatalf("cycle = %d %s", cycleResponse.Code, cycleResponse.Body.String())
	}

	reloaded := getJSON(t, router, "/organograms/"+organogram["id"].(string)+"/positions", hr)
	if len(reloaded) != 3 || reloaded[0]["title"] == "" {
		t.Fatalf("reloaded = %#v", reloaded)
	}
	removeManager := httptest.NewRequest(http.MethodDelete, "/organograms/"+organogram["id"].(string)+"/positions/"+manager["id"].(string), nil)
	removeManager.Header.Set("Authorization", "Bearer "+sign(t, hr))
	removeResponse := httptest.NewRecorder()
	router.ServeHTTP(removeResponse, removeManager)
	if removeResponse.Code != http.StatusNoContent {
		t.Fatalf("delete manager = %d %s", removeResponse.Code, removeResponse.Body.String())
	}
	removeDirector := httptest.NewRequest(http.MethodDelete, "/organograms/"+organogram["id"].(string)+"/positions/"+director["id"].(string), nil)
	removeDirector.Header.Set("Authorization", "Bearer "+sign(t, hr))
	blocked := httptest.NewRecorder()
	router.ServeHTTP(blocked, removeDirector)
	if blocked.Code != http.StatusBadRequest || !strings.Contains(blocked.Body.String(), "Move the positions") {
		t.Fatalf("delete director = %d %s", blocked.Code, blocked.Body.String())
	}

	publish := httptest.NewRequest(http.MethodPost, "/organograms/"+organogram["id"].(string)+"/publish", nil)
	publish.Header.Set("Authorization", "Bearer "+sign(t, hr))
	published := httptest.NewRecorder()
	router.ServeHTTP(published, publish)
	if published.Code != http.StatusOK || !strings.Contains(published.Body.String(), `"status":"published"`) {
		t.Fatalf("publish = %d %s", published.Code, published.Body.String())
	}
}

func TestOrganogramImportValidatesFilesAndPermissions(t *testing.T) {
	router := NewRouter(nil, []byte(secret), seeded(), &recordingPublisher{})
	hr := token.Claims{PersonID: adaID, OrganisationID: organisationID, Roles: []string{"hr"}}
	employee := token.Claims{PersonID: eliID, OrganisationID: organisationID, Roles: []string{"employee"}}
	manager := token.Claims{PersonID: mayaID, OrganisationID: organisationID, Roles: []string{"manager"}, ManagerOf: []string{eliID}}
	organogram := createChart(t, router, hr, "Import chart")
	chartID := organogram["id"].(string)

	denied := uploadChartFile(t, router, employee, chartID, "roles.xlsx", xlsxSheet(t, [][]string{{"title", "reports_to", "department"}, {"Lead", "", "Sales"}}), "", "", "")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("employee upload = %d", denied.Code)
	}
	managerUpload := uploadChartFile(t, router, manager, chartID, "roles.xlsx", xlsxSheet(t, [][]string{{"title"}, {"Lead"}}), "", "", "")
	if managerUpload.Code != http.StatusForbidden {
		t.Fatalf("manager upload = %d", managerUpload.Code)
	}
	template := httptest.NewRequest(http.MethodGet, "/organogram-template", nil)
	template.Header.Set("Authorization", "Bearer "+sign(t, manager))
	templateResponse := httptest.NewRecorder()
	router.ServeHTTP(templateResponse, template)
	if templateResponse.Code != http.StatusOK || !bytes.HasPrefix(templateResponse.Body.Bytes(), []byte("PK")) {
		t.Fatalf("template = %d %d bytes", templateResponse.Code, templateResponse.Body.Len())
	}

	invalid := uploadChartFile(t, router, hr, chartID, "notes.txt", []byte("hello"), "", "", "")
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid = %d %s", invalid.Code, invalid.Body.String())
	}
	png := uploadChartFile(t, router, hr, chartID, "chart.png", []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, "", "", "")
	pngResult := decodeImport(t, png)
	if png.Code != http.StatusCreated || pngResult["file"].(map[string]any)["status"] != FileNeedsReview {
		t.Fatalf("png = %d %#v", png.Code, pngResult)
	}
	xls := uploadChartFile(t, router, hr, chartID, "chart.xls", []byte{0xd0, 0xcf, 0x11, 0xe0, 0x00}, "", "", "")
	xlsResult := decodeImport(t, xls)
	if xls.Code != http.StatusCreated || xlsResult["file"].(map[string]any)["status"] != FileRejected || !strings.Contains(xlsResult["file"].(map[string]any)["detail"].(string), "XLSX") {
		t.Fatalf("xls = %#v", xlsResult)
	}
	if len(getJSON(t, router, "/organograms/"+chartID+"/positions", hr)) != 0 {
		t.Fatal("visual and xls files created positions")
	}

	imported := uploadChartFile(t, router, hr, chartID, "roles.xlsx", xlsxSheet(t, [][]string{
		{"title", "reports_to", "department"},
		{"Managing Director", "", ""},
		{"Sales Manager", "Managing Director", "Sales"},
		{"", "Managing Director", "Sales"},
		{"Analyst", "Managing Director", "Missing"},
	}), "", "", "")
	importedResult := decodeImport(t, imported)
	if imported.Code != http.StatusCreated || importedResult["accepted"] != float64(2) || len(importedResult["rejected"].([]any)) != 2 {
		t.Fatalf("import = %d %#v", imported.Code, importedResult)
	}
	positions := getJSON(t, router, "/organograms/"+chartID+"/positions", manager)
	if len(positions) != 2 {
		t.Fatalf("positions = %#v", positions)
	}
	files := getJSON(t, router, "/organograms/"+chartID+"/files", manager)
	if len(files) != 3 {
		t.Fatalf("files = %#v", files)
	}
	download := httptest.NewRequest(http.MethodGet, "/organograms/"+chartID+"/files/"+pngResult["file"].(map[string]any)["id"].(string), nil)
	download.Header.Set("Authorization", "Bearer "+sign(t, manager))
	downloadResponse := httptest.NewRecorder()
	router.ServeHTTP(downloadResponse, download)
	if downloadResponse.Code != http.StatusOK || !bytes.HasPrefix(downloadResponse.Body.Bytes(), []byte{0x89, 'P'}) {
		t.Fatalf("download = %d", downloadResponse.Code)
	}
	employeeDownload := httptest.NewRequest(http.MethodGet, "/organograms/"+chartID+"/files/"+pngResult["file"].(map[string]any)["id"].(string), nil)
	employeeDownload.Header.Set("Authorization", "Bearer "+sign(t, employee))
	employeeDownloadResponse := httptest.NewRecorder()
	router.ServeHTTP(employeeDownloadResponse, employeeDownload)
	if employeeDownloadResponse.Code != http.StatusForbidden {
		t.Fatalf("employee download = %d", employeeDownloadResponse.Code)
	}

	mapped := uploadChartFile(t, router, hr, chartID, "mapped.xlsx", xlsxSheet(t, [][]string{{"Job", "Boss"}, {"Regional Lead", "Managing Director"}}), "Job", "Boss", "")
	mappedResult := decodeImport(t, mapped)
	if mapped.Code != http.StatusCreated || mappedResult["accepted"] != float64(1) {
		t.Fatalf("mapped = %d %#v", mapped.Code, mappedResult)
	}
}

func createChart(t *testing.T, router http.Handler, claims token.Claims, name string) map[string]any {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/organograms", strings.NewReader(`{"name":"`+name+`"}`))
	request.Header.Set("Authorization", "Bearer "+sign(t, claims))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create chart = %d %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func createChartPosition(t *testing.T, router http.Handler, claims token.Claims, organogramID, body string) map[string]any {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/organograms/"+organogramID+"/positions", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+sign(t, claims))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create position = %d %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func patchPosition(t *testing.T, router http.Handler, claims token.Claims, organogramID, positionID, body string) map[string]any {
	t.Helper()
	request := httptest.NewRequest(http.MethodPatch, "/organograms/"+organogramID+"/positions/"+positionID, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+sign(t, claims))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("patch = %d %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func uploadChartFile(t *testing.T, router http.Handler, claims token.Claims, organogramID, name string, body []byte, titleColumn, reportsColumn, departmentColumn string) *httptest.ResponseRecorder {
	t.Helper()
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatal(err)
	}
	if titleColumn != "" {
		if err := writer.WriteField("titleColumn", titleColumn); err != nil {
			t.Fatal(err)
		}
		if err := writer.WriteField("reportsToColumn", reportsColumn); err != nil {
			t.Fatal(err)
		}
	}
	if departmentColumn != "" {
		if err := writer.WriteField("departmentColumn", departmentColumn); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/organograms/"+organogramID+"/files", &buffer)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+sign(t, claims))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func decodeImport(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if response.Code != http.StatusCreated {
		t.Fatalf("import status = %d %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func xlsxSheet(t *testing.T, rows [][]string) []byte {
	t.Helper()
	var sheet strings.Builder
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for rowIndex, row := range rows {
		sheet.WriteString(`<row r="` + itoa(rowIndex+1) + `">`)
		for column, value := range row {
			ref := string(rune('A'+column)) + itoa(rowIndex+1)
			sheet.WriteString(`<c r="` + ref + `" t="inlineStr"><is><t>` + value + `</t></is></c>`)
		}
		sheet.WriteString(`</row>`)
	}
	sheet.WriteString(`</sheetData></worksheet>`)
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	files := map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Organogram" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml":   sheet.String(),
	}
	for name, contents := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestAttachmentNameDropsHeaderInjection(t *testing.T) {
	if strings.ContainsAny(safeAttachmentName("chart\r\nX: y.xlsx"), "\r\n\";\\") {
		t.Fatal("header characters remained")
	}
	if safeAttachmentName("\"") != "download" {
		t.Fatal("empty name was not replaced")
	}
}
