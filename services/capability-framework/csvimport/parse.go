package csvimport

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	ProposalDuplicateCode = "duplicate_capability_code"
	ProposalMissingLevel  = "missing_required_level"
)

var RequiredColumns = []string{
	"framework version",
	"department",
	"role code",
	"role name",
	"capability code",
	"capability name",
	"capability type",
	"definition",
	"required level",
	"proficiency levels",
	"behavioural indicators",
	"assessment methods",
	"learning asset ids",
	"practice scenario ids",
	"workplace mission",
	"weight",
	"governance status",
}

type Row struct {
	RowNumber             int
	FrameworkVersion      string
	Department            string
	RoleCode              string
	RoleName              string
	CapabilityCode        string
	CapabilityName        string
	CapabilityType        string
	Definition            string
	RequiredLevel         string
	RequiredLevelValue    *int
	ProficiencyLevels     string
	BehaviouralIndicators string
	AssessmentMethods     string
	LearningAssetIDs      string
	PracticeScenarioIDs   string
	WorkplaceMission      string
	Weight                string
	GovernanceStatus      string
	IndicatorStatements   []string
	Indicators            []PlacedIndicator
}

type PlacedIndicator struct {
	Level     int
	Statement string
}

type RowError struct {
	Row     int
	Message string
}

type Proposal struct {
	Kind           string
	CapabilityCode string
	Message        string
}

type ParseResult struct {
	MissingColumns []string
	RowErrors      []RowError
	Rows           []Row
	Proposals      []Proposal
}

func Parse(reader io.Reader) (ParseResult, error) {
	records, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		return ParseResult{}, err
	}
	if len(records) == 0 {
		return ParseResult{MissingColumns: append([]string{}, RequiredColumns...)}, nil
	}
	header := map[string]int{}
	for index, column := range records[0] {
		header[normalize(column)] = index
	}
	missing := make([]string, 0)
	for _, column := range RequiredColumns {
		if _, ok := header[column]; !ok {
			missing = append(missing, column)
		}
	}
	if len(missing) > 0 {
		return ParseResult{MissingColumns: missing}, nil
	}

	result := ParseResult{RowErrors: []RowError{}, Rows: []Row{}, Proposals: []Proposal{}}
	for offset, record := range records[1:] {
		rowNumber := offset + 2
		row, errors := readRow(rowNumber, record, header)
		if len(errors) > 0 {
			result.RowErrors = append(result.RowErrors, errors...)
			continue
		}
		result.Rows = append(result.Rows, row)
	}
	result.Proposals = check(result.Rows)
	return result, nil
}

func readRow(rowNumber int, record []string, header map[string]int) (Row, []RowError) {
	value := func(column string) string {
		index := header[column]
		if index >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[index])
	}
	row := Row{
		RowNumber:             rowNumber,
		FrameworkVersion:      value("framework version"),
		Department:            value("department"),
		RoleCode:              value("role code"),
		RoleName:              value("role name"),
		CapabilityCode:        value("capability code"),
		CapabilityName:        value("capability name"),
		CapabilityType:        value("capability type"),
		Definition:            value("definition"),
		RequiredLevel:         value("required level"),
		ProficiencyLevels:     value("proficiency levels"),
		BehaviouralIndicators: value("behavioural indicators"),
		AssessmentMethods:     value("assessment methods"),
		LearningAssetIDs:      value("learning asset ids"),
		PracticeScenarioIDs:   value("practice scenario ids"),
		WorkplaceMission:      value("workplace mission"),
		Weight:                value("weight"),
		GovernanceStatus:      value("governance status"),
	}
	errors := make([]RowError, 0)
	require := func(field, label string) {
		if field == "" {
			errors = append(errors, RowError{Row: rowNumber, Message: label + " is required"})
		}
	}
	require(row.FrameworkVersion, "framework version")
	require(row.Department, "department")
	require(row.RoleCode, "role code")
	require(row.RoleName, "role name")
	require(row.CapabilityCode, "capability code")
	require(row.CapabilityName, "capability name")
	require(row.CapabilityType, "capability type")
	require(row.Definition, "definition")
	if row.RequiredLevel != "" {
		level, err := strconv.Atoi(row.RequiredLevel)
		if err != nil || level < 0 || level > 4 {
			errors = append(errors, RowError{Row: rowNumber, Message: "required level must be a whole number from 0 to 4"})
		} else {
			row.RequiredLevelValue = &level
		}
	}
	if len(errors) > 0 {
		return Row{}, errors
	}
	placed, placeErrors := placeIndicators(row)
	if len(placeErrors) > 0 {
		return Row{}, placeErrors
	}
	row.IndicatorStatements = statementsOf(placed)
	row.Indicators = placed
	return row, nil
}

func check(rows []Row) []Proposal {
	proposals := make([]Proposal, 0)
	seen := map[string]int{}
	for _, row := range rows {
		seen[row.CapabilityCode]++
		if row.RequiredLevel == "" {
			proposals = append(proposals, Proposal{
				Kind:           ProposalMissingLevel,
				CapabilityCode: row.CapabilityCode,
				Message:        fmt.Sprintf("Row %d is missing a required level for capability code %s", row.RowNumber, row.CapabilityCode),
			})
		}
	}
	for _, row := range rows {
		if seen[row.CapabilityCode] > 1 {
			proposals = append(proposals, Proposal{
				Kind:           ProposalDuplicateCode,
				CapabilityCode: row.CapabilityCode,
				Message:        fmt.Sprintf("Capability code %s is duplicated", row.CapabilityCode),
			})
			seen[row.CapabilityCode] = 0
		}
	}
	return proposals
}

func placeIndicators(row Row) ([]PlacedIndicator, []RowError) {
	levels := splitList(row.ProficiencyLevels)
	statements := splitList(row.BehaviouralIndicators)
	if len(statements) == 0 {
		return []PlacedIndicator{}, nil
	}
	if len(levels) == len(statements) {
		placed := make([]PlacedIndicator, 0, len(statements))
		for index, statement := range statements {
			level, err := strconv.Atoi(levels[index])
			if err != nil || level < 0 || level > 4 {
				return nil, []RowError{{Row: row.RowNumber, Message: "proficiency levels must be whole numbers from 0 to 4"}}
			}
			placed = append(placed, PlacedIndicator{Level: level, Statement: statement})
		}
		return placed, nil
	}
	if row.RequiredLevelValue == nil {
		return []PlacedIndicator{}, nil
	}
	placed := make([]PlacedIndicator, 0, len(statements))
	for _, statement := range statements {
		placed = append(placed, PlacedIndicator{Level: *row.RequiredLevelValue, Statement: statement})
	}
	return placed, nil
}

func statementsOf(placed []PlacedIndicator) []string {
	statements := make([]string, 0, len(placed))
	for _, indicator := range placed {
		statements = append(statements, indicator.Statement)
	}
	return statements
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '|' || r == ';'
	})
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}

func normalize(column string) string {
	column = strings.TrimPrefix(column, "\uFEFF")
	return strings.ToLower(strings.TrimSpace(column))
}
