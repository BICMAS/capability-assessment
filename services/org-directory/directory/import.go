package directory

import "strings"

const maxImportRows = 500

type preparedImport struct {
	mediaType string
	status    string
	detail    string
	headers   []string
	rejected  []ImportRejection
	positions []plannedPosition
}

type plannedPosition struct {
	Title           string
	DepartmentID    string
	ReportsToTitle  string
	ReportsToExists bool
}

func prepareImport(fileName string, body []byte, mapping SheetMapping, existing []Position, departments []Department) (preparedImport, error) {
	mediaType, kind, err := classifyOrganogramFile(fileName, body)
	if err != nil {
		return preparedImport{}, err
	}
	prepared := preparedImport{mediaType: mediaType, rejected: []ImportRejection{}, headers: []string{}}
	switch kind {
	case "png", "jpeg", "pdf":
		prepared.status = FileNeedsReview
		prepared.detail = "The file is stored. No extraction service is configured, so this image or PDF was not turned into positions."
		return prepared, nil
	case "xls":
		prepared.status = FileRejected
		prepared.detail = "This is an XLS workbook. It was stored and not imported. Save it as XLSX and upload that file."
		return prepared, nil
	}
	table, err := readXLSX(body)
	if err != nil {
		prepared.status = FileRejected
		prepared.detail = "The spreadsheet could not be read."
		return prepared, nil
	}
	accepted, rejected, headers, failure := planSheet(existing, departments, table, mapping)
	prepared.headers = headers
	prepared.rejected = rejected
	if failure != "" {
		prepared.status = FileRejected
		prepared.detail = failure
		return prepared, nil
	}
	prepared.positions = accepted
	if len(accepted) == 0 {
		prepared.status = FileRejected
		prepared.detail = "No positions were imported."
		return prepared, nil
	}
	prepared.status = FileImported
	prepared.detail = importDetail(len(accepted), len(rejected))
	return prepared, nil
}

func importDetail(accepted, rejected int) string {
	if rejected == 0 {
		return "Imported " + itoa(accepted) + " positions."
	}
	return "Imported " + itoa(accepted) + " positions. " + itoa(rejected) + " rows were rejected."
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

func planSheet(existing []Position, departments []Department, table [][]string, mapping SheetMapping) ([]plannedPosition, []ImportRejection, []string, string) {
	if len(table) == 0 {
		return nil, []ImportRejection{}, []string{}, "The spreadsheet has no header row."
	}
	headers := table[0]
	titleColumn := mappedColumn(headers, mapping.Title, "title", "job_title", "position", "position_title")
	reportsColumn := mappedColumn(headers, mapping.ReportsTo, "reports_to", "reports_to_title", "manager", "parent")
	departmentColumn := mappedColumn(headers, mapping.Department, "department", "department_name", "unit")
	if mapping.Title != "" && titleColumn < 0 {
		return nil, []ImportRejection{}, headers, "The title column is not in this spreadsheet."
	}
	if titleColumn < 0 {
		return nil, []ImportRejection{}, headers, "Choose the column that contains the position title."
	}
	rejected := []ImportRejection{}
	type candidate struct {
		row        int
		title      string
		reportsTo  string
		department string
	}
	candidates := []candidate{}
	seen := map[string]int{}
	dataRows := 0
	for offset, row := range table[1:] {
		if dataRows >= maxImportRows {
			rejected = append(rejected, ImportRejection{Row: offset + 2, Message: "Only the first 500 position rows can be imported."})
			continue
		}
		title := cell(row, titleColumn)
		reportsTo := cell(row, reportsColumn)
		department := cell(row, departmentColumn)
		if title == "" && reportsTo == "" && department == "" {
			continue
		}
		dataRows++
		rowNumber := offset + 2
		if title == "" {
			rejected = append(rejected, ImportRejection{Row: rowNumber, Message: "Title is required."})
			continue
		}
		key := strings.ToLower(title)
		if _, duplicate := seen[key]; duplicate {
			rejected = append(rejected, ImportRejection{Row: rowNumber, Message: "Title is duplicated in this file."})
			continue
		}
		seen[key] = rowNumber
		if department != "" && departmentID(departments, department) == "" {
			rejected = append(rejected, ImportRejection{Row: rowNumber, Message: "Department is not in this organisation."})
			continue
		}
		candidates = append(candidates, candidate{row: rowNumber, title: title, reportsTo: reportsTo, department: department})
	}
	existingIDs := titleIndex(existing)
	blocked := map[string]bool{}
	for _, row := range candidates {
		if strings.EqualFold(row.title, row.reportsTo) && row.reportsTo != "" {
			rejected = append(rejected, ImportRejection{Row: row.row, Message: "Reporting relationship is circular."})
			blocked[strings.ToLower(row.title)] = true
		}
	}
	changed := true
	for changed {
		changed = false
		for _, row := range candidates {
			key := strings.ToLower(row.title)
			if blocked[key] || row.reportsTo == "" || strings.EqualFold(row.title, row.reportsTo) {
				continue
			}
			parent := strings.ToLower(row.reportsTo)
			_, inFile := seen[parent]
			_, exists := existingIDs[parent]
			if blocked[parent] && inFile {
				rejected = append(rejected, ImportRejection{Row: row.row, Message: "Reports to a row that was rejected."})
				blocked[key] = true
				changed = true
				continue
			}
			if !inFile && !exists {
				rejected = append(rejected, ImportRejection{Row: row.row, Message: "Reports to an unknown position."})
				blocked[key] = true
				changed = true
				continue
			}
			if !inFile && len(existingIDs[parent]) > 1 {
				rejected = append(rejected, ImportRejection{Row: row.row, Message: "Reports to more than one existing position."})
				blocked[key] = true
				changed = true
			}
		}
	}
	accepted := []plannedPosition{}
	for _, row := range candidates {
		if blocked[strings.ToLower(row.title)] {
			continue
		}
		accepted = append(accepted, plannedPosition{
			Title:           row.title,
			DepartmentID:    departmentID(departments, row.department),
			ReportsToTitle:  row.reportsTo,
			ReportsToExists: row.reportsTo != "" && seen[strings.ToLower(row.reportsTo)] == 0,
		})
	}
	circular := cyclicTitles(accepted)
	if len(circular) > 0 {
		kept := []plannedPosition{}
		for _, row := range candidates {
			if circular[strings.ToLower(row.title)] {
				rejected = append(rejected, ImportRejection{Row: row.row, Message: "Reporting relationship is circular."})
			}
		}
		for _, position := range accepted {
			if !circular[strings.ToLower(position.Title)] {
				kept = append(kept, position)
			}
		}
		accepted = kept
	}
	return orderPositions(accepted), rejected, headers, ""
}

func mappedColumn(headers []string, requested string, names ...string) int {
	if requested != "" {
		return headerIndex(headers, requested)
	}
	for _, name := range names {
		if index := headerIndex(headers, name); index >= 0 {
			return index
		}
	}
	return -1
}

func headerIndex(headers []string, name string) int {
	want := normaliseHeader(name)
	for index, header := range headers {
		if normaliseHeader(header) == want {
			return index
		}
	}
	return -1
}

func normaliseHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.ReplaceAll(value, " ", "_")
}

func cell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func departmentID(departments []Department, name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	for _, department := range departments {
		if strings.EqualFold(department.Name, strings.TrimSpace(name)) {
			return department.ID
		}
	}
	return ""
}

func resolveParent(existing map[string][]string, created map[string]string, position plannedPosition) (string, error) {
	if position.ReportsToTitle == "" {
		return "", nil
	}
	key := strings.ToLower(position.ReportsToTitle)
	if position.ReportsToExists {
		ids := existing[key]
		if len(ids) != 1 {
			return "", ErrInvalid
		}
		return ids[0], nil
	}
	id, ok := created[key]
	if !ok {
		return "", ErrInvalid
	}
	return id, nil
}

func titleIndex(positions []Position) map[string][]string {
	index := map[string][]string{}
	for _, position := range positions {
		key := strings.ToLower(strings.TrimSpace(position.Title))
		index[key] = append(index[key], position.ID)
	}
	return index
}

func cyclicTitles(positions []plannedPosition) map[string]bool {
	next := map[string]string{}
	for _, position := range positions {
		if position.ReportsToTitle == "" || position.ReportsToExists {
			continue
		}
		next[strings.ToLower(position.Title)] = strings.ToLower(position.ReportsToTitle)
	}
	blocked := map[string]bool{}
	for start := range next {
		seen := map[string]bool{}
		current := start
		for current != "" {
			if seen[current] {
				for item := range seen {
					blocked[item] = true
				}
				break
			}
			seen[current] = true
			current = next[current]
		}
	}
	return blocked
}

func orderPositions(positions []plannedPosition) []plannedPosition {
	pending := append([]plannedPosition{}, positions...)
	ordered := make([]plannedPosition, 0, len(pending))
	ready := map[string]bool{}
	for len(pending) > 0 {
		next := []plannedPosition{}
		progress := false
		for _, position := range pending {
			parentReady := position.ReportsToTitle == "" || position.ReportsToExists || ready[strings.ToLower(position.ReportsToTitle)]
			if parentReady {
				ordered = append(ordered, position)
				ready[strings.ToLower(position.Title)] = true
				progress = true
				continue
			}
			next = append(next, position)
		}
		if !progress {
			return append(ordered, next...)
		}
		pending = next
	}
	return ordered
}

func reportingCycle(positions []Position, positionID, reportsTo string) bool {
	if reportsTo == "" {
		return false
	}
	if reportsTo == positionID {
		return true
	}
	seen := map[string]bool{}
	current := reportsTo
	for current != "" {
		if current == positionID || seen[current] {
			return true
		}
		seen[current] = true
		next := ""
		found := false
		for _, position := range positions {
			if position.ID == current {
				next = position.ReportsToPositionID
				found = true
				break
			}
		}
		if !found {
			return false
		}
		current = next
	}
	return false
}

func positionHasReports(positions []Position, positionID string) bool {
	for _, position := range positions {
		if position.ReportsToPositionID == positionID {
			return true
		}
	}
	return false
}
