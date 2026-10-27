package roleprofile

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

type Memory struct {
	mu           sync.Mutex
	roles        []Role
	skills       []Skill
	profiles     []Profile
	versions     []Version
	comments     []Comment
	events       []AuditEvent
	contexts     []ContextRecord
	questions    []Question
	jobs         []Job
	frameworks   map[string]map[string]struct{}
	publications [][]byte
}

func NewMemory() *Memory {
	return &Memory{frameworks: map[string]map[string]struct{}{}}
}

// NoteFrameworkVersion records a framework version owned by the capability framework domain.
func (m *Memory) NoteFrameworkVersion(organisationID, versionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.frameworks[organisationID] == nil {
		m.frameworks[organisationID] = map[string]struct{}{}
	}
	m.frameworks[organisationID][versionID] = struct{}{}
}

func (m *Memory) CreateRole(_ context.Context, organisationID, name string) (Role, error) {
	name = strings.TrimSpace(name)
	if organisationID == "" || name == "" {
		return Role{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, role := range m.roles {
		if role.OrganisationID == organisationID && role.Name == name {
			return Role{}, ErrDuplicate
		}
	}
	id, err := newID()
	if err != nil {
		return Role{}, err
	}
	role := Role{ID: id, OrganisationID: organisationID, Name: name}
	m.roles = append(m.roles, role)
	return role, nil
}

func (m *Memory) CreateSkill(_ context.Context, organisationID string, skill Skill) (Skill, error) {
	skill.Name = strings.TrimSpace(skill.Name)
	skill.Definition = strings.TrimSpace(skill.Definition)
	if organisationID == "" || skill.Name == "" || skill.Definition == "" || !validCategory(skill.Category) {
		return Skill{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.skills {
		if existing.OrganisationID == organisationID && existing.Name == skill.Name {
			return Skill{}, ErrDuplicate
		}
	}
	id, err := newID()
	if err != nil {
		return Skill{}, err
	}
	created := Skill{ID: id, OrganisationID: organisationID, Name: skill.Name, Category: skill.Category, Definition: skill.Definition}
	m.skills = append(m.skills, created)
	return created, nil
}

func (m *Memory) CreateDraft(_ context.Context, organisationID, roleID, actorID, frameworkVersionID string) (Profile, error) {
	if organisationID == "" || actorID == "" {
		return Profile{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	role, ok := m.role(organisationID, roleID)
	if !ok {
		return Profile{}, ErrNotFound
	}
	if frameworkVersionID != "" {
		if _, known := m.frameworks[organisationID][frameworkVersionID]; !known {
			return Profile{}, ErrNotFound
		}
	}
	for _, profile := range m.profiles {
		if profile.OrganisationID == organisationID && profile.RoleID == roleID {
			return Profile{}, ErrDuplicate
		}
	}
	profileID, err := newID()
	if err != nil {
		return Profile{}, err
	}
	versionID, err := newID()
	if err != nil {
		return Profile{}, err
	}
	now := clock()
	version := blankVersion(versionID, profileID, organisationID, frameworkVersionID, 1, now)
	profile := Profile{
		ID:                profileID,
		OrganisationID:    organisationID,
		RoleID:            role.ID,
		RoleName:          role.Name,
		CreatedByPersonID: actorID,
		CreatedAt:         now,
		Version:           version,
	}
	m.profiles = append(m.profiles, profile)
	m.versions = append(m.versions, version)
	if err := m.audit(organisationID, profileID, versionID, actorID, ActionCreated, now); err != nil {
		return Profile{}, err
	}
	return m.profileView(profile, version), nil
}

func (m *Memory) GetProfile(_ context.Context, organisationID, profileID string) (Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current(organisationID, profileID)
}

func (m *Memory) UpdateContext(_ context.Context, organisationID, versionID, actorID string, expected int, details Context) (Version, error) {
	if err := validContext(details); err != nil {
		return Version{}, err
	}
	if actorID == "" {
		return Version{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	version, err := m.editable(organisationID, versionID, expected)
	if err != nil {
		return Version{}, err
	}
	sources, err := m.sources(details.Sources)
	if err != nil {
		return Version{}, err
	}
	version.Purpose = strings.TrimSpace(details.Purpose)
	version.Responsibilities = strings.TrimSpace(details.Responsibilities)
	version.BusinessOutcomes = strings.TrimSpace(details.BusinessOutcomes)
	version.DecisionScope = strings.TrimSpace(details.DecisionScope)
	version.Industry = strings.TrimSpace(details.Industry)
	version.Region = strings.TrimSpace(details.Region)
	version.ReportingLevel = details.ReportingLevel
	version.TeamSize = copyInt(details.TeamSize)
	version.Sources = sources
	version.Revision++
	if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionContext, clock()); err != nil {
		return Version{}, err
	}
	return cloneVersion(*version), nil
}

func (m *Memory) AddSkill(_ context.Context, organisationID, versionID, actorID string, expected int, input SkillInput) (Version, error) {
	if err := validSkillInput(input); err != nil {
		return Version{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	version, err := m.editable(organisationID, versionID, expected)
	if err != nil {
		return Version{}, err
	}
	skill, ok := m.skill(organisationID, input.SkillID)
	if !ok || skill.Category != input.Category {
		if !ok {
			return Version{}, ErrNotFound
		}
		return Version{}, ErrInvalid
	}
	for _, existing := range version.Skills {
		if existing.SkillID == skill.ID {
			return Version{}, ErrDuplicate
		}
	}
	id, err := newID()
	if err != nil {
		return Version{}, err
	}
	version.Skills = append(version.Skills, SkillAssignment{
		ID:                  id,
		SkillID:             skill.ID,
		SkillName:           skill.Name,
		Category:            skill.Category,
		Definition:          strings.TrimSpace(input.Definition),
		RequiredProficiency: input.RequiredProficiency,
		Priority:            input.Priority,
		Rationale:           strings.TrimSpace(input.Rationale),
		Provenance:          input.Provenance,
		Position:            len(version.Skills) + 1,
	})
	version.Revision++
	if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionSkillAdded, clock()); err != nil {
		return Version{}, err
	}
	return cloneVersion(*version), nil
}

func (m *Memory) UpdateSkill(_ context.Context, organisationID, versionID, assignmentID, actorID string, expected int, edit SkillEdit) (Version, error) {
	if err := validSkillEdit(edit); err != nil {
		return Version{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	version, err := m.editable(organisationID, versionID, expected)
	if err != nil {
		return Version{}, err
	}
	index := skillIndex(version.Skills, assignmentID)
	if index < 0 {
		return Version{}, ErrNotFound
	}
	skill := version.Skills[index]
	skill.Definition = strings.TrimSpace(edit.Definition)
	skill.RequiredProficiency = edit.RequiredProficiency
	skill.Priority = edit.Priority
	skill.Rationale = strings.TrimSpace(edit.Rationale)
	if skill.Provenance == ProvenanceAISuggested {
		skill.Provenance = ProvenanceHRAdjusted
	}
	version.Skills[index] = skill
	version.Revision++
	if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionSkillUpdated, clock()); err != nil {
		return Version{}, err
	}
	return cloneVersion(*version), nil
}

func (m *Memory) RemoveSkill(_ context.Context, organisationID, versionID, assignmentID, actorID string, expected int) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	version, err := m.editable(organisationID, versionID, expected)
	if err != nil {
		return Version{}, err
	}
	index := skillIndex(version.Skills, assignmentID)
	if index < 0 {
		return Version{}, ErrNotFound
	}
	version.Skills = append(version.Skills[:index], version.Skills[index+1:]...)
	for i := range version.Skills {
		version.Skills[i].Position = i + 1
	}
	version.Revision++
	if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionSkillRemoved, clock()); err != nil {
		return Version{}, err
	}
	return cloneVersion(*version), nil
}

func (m *Memory) ReorderSkills(_ context.Context, organisationID, versionID, actorID string, expected int, assignmentIDs []string) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	version, err := m.editable(organisationID, versionID, expected)
	if err != nil {
		return Version{}, err
	}
	if !permutation(version.Skills, assignmentIDs) {
		return Version{}, ErrInvalid
	}
	ordered := make([]SkillAssignment, 0, len(assignmentIDs))
	for position, id := range assignmentIDs {
		index := skillIndex(version.Skills, id)
		skill := version.Skills[index]
		skill.Position = position + 1
		ordered = append(ordered, skill)
	}
	version.Skills = ordered
	version.Revision++
	if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionReordered, clock()); err != nil {
		return Version{}, err
	}
	return cloneVersion(*version), nil
}

func (m *Memory) ListVersions(_ context.Context, organisationID, profileID string) ([]Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.identity(organisationID, profileID); !ok {
		return nil, ErrNotFound
	}
	versions := make([]Version, 0)
	for _, version := range m.versions {
		if version.OrganisationID == organisationID && version.ProfileID == profileID {
			versions = append(versions, cloneVersion(version))
		}
	}
	return versions, nil
}

func (m *Memory) Compare(_ context.Context, organisationID, profileID, fromID, toID string) (Comparison, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.identity(organisationID, profileID); !ok {
		return Comparison{}, ErrNotFound
	}
	from, ok := m.version(organisationID, fromID)
	if !ok || from.ProfileID != profileID {
		return Comparison{}, ErrNotFound
	}
	to, ok := m.version(organisationID, toID)
	if !ok || to.ProfileID != profileID {
		return Comparison{}, ErrNotFound
	}
	return compareVersions(cloneVersion(*from), cloneVersion(*to)), nil
}

func (m *Memory) Submit(_ context.Context, organisationID, versionID, actorID string, expected int) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	version, err := m.guard(organisationID, versionID, expected, StatusDraft)
	if err != nil {
		return Version{}, err
	}
	if err := readyForReview(*version); err != nil {
		return Version{}, err
	}
	version.Status = StatusInReview
	version.Revision++
	if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionSubmitted, clock()); err != nil {
		return Version{}, err
	}
	return cloneVersion(*version), nil
}

func (m *Memory) Publish(_ context.Context, organisationID, versionID, actorID string, expected int) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	version, err := m.guard(organisationID, versionID, expected, StatusInReview)
	if err != nil {
		return Version{}, err
	}
	if err := readyForReview(*version); err != nil {
		return Version{}, err
	}
	occurred := clock()
	payload, err := roleProfilePublishedPayload(organisationID, version.ProfileID, version.ID, actorID, version.VersionNumber, occurred)
	if err != nil {
		return Version{}, err
	}
	version.Status = StatusPublished
	version.Revision++
	if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionPublished, occurred); err != nil {
		return Version{}, err
	}
	m.publications = append(m.publications, payload)
	return cloneVersion(*version), nil
}

func (m *Memory) Publications() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([][]byte, len(m.publications))
	for i, payload := range m.publications {
		copied[i] = append([]byte(nil), payload...)
	}
	return copied
}

func (m *Memory) NextDraft(_ context.Context, organisationID, versionID, actorID string, expected int) (Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	source, err := m.guard(organisationID, versionID, expected, StatusPublished)
	if err != nil {
		return Version{}, err
	}
	if source.VersionNumber != m.maxNumber(source.ProfileID) {
		return Version{}, ErrConflict
	}
	id, err := newID()
	if err != nil {
		return Version{}, err
	}
	now := clock()
	next := cloneVersion(*source)
	next.ID = id
	next.VersionNumber = source.VersionNumber + 1
	next.Revision = 1
	next.Status = StatusDraft
	next.CreatedAt = now
	for i := range next.Sources {
		sourceID, err := newID()
		if err != nil {
			return Version{}, err
		}
		next.Sources[i].ID = sourceID
	}
	for i := range next.Skills {
		skillID, err := newID()
		if err != nil {
			return Version{}, err
		}
		next.Skills[i].ID = skillID
	}
	m.versions = append(m.versions, next)
	if err := m.audit(organisationID, source.ProfileID, next.ID, actorID, ActionNextDraft, now); err != nil {
		return Version{}, err
	}
	return cloneVersion(next), nil
}

func (m *Memory) Comment(_ context.Context, organisationID, versionID, actorID, body string) (Comment, error) {
	body = strings.TrimSpace(body)
	if actorID == "" || body == "" {
		return Comment{}, ErrInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	version, ok := m.version(organisationID, versionID)
	if !ok {
		return Comment{}, ErrNotFound
	}
	if version.Status != StatusInReview {
		return Comment{}, ErrImmutable
	}
	id, err := newID()
	if err != nil {
		return Comment{}, err
	}
	comment := Comment{ID: id, ProfileVersionID: versionID, AuthorPersonID: actorID, Body: body, CreatedAt: clock()}
	m.comments = append(m.comments, comment)
	if err := m.audit(organisationID, version.ProfileID, versionID, actorID, ActionComment, comment.CreatedAt); err != nil {
		return Comment{}, err
	}
	return comment, nil
}

func (m *Memory) History(_ context.Context, organisationID, profileID string) (History, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.identity(organisationID, profileID); !ok {
		return History{}, ErrNotFound
	}
	events := make([]AuditEvent, 0)
	for _, event := range m.events {
		if event.ProfileVersionID == "" {
			continue
		}
		if m.eventProfile(event) == profileID {
			events = append(events, event)
		}
	}
	comments := make([]Comment, 0)
	for _, comment := range m.comments {
		version, ok := m.version(organisationID, comment.ProfileVersionID)
		if ok && version.ProfileID == profileID {
			comments = append(comments, comment)
		}
	}
	return History{ProfileID: profileID, Events: events, Comments: comments}, nil
}

func (m *Memory) AttachAIDraft(_ context.Context, organisationID, versionID, actorID string, draft AIDraft) (Version, error) {
	if err := validAIDraft(draft); err != nil {
		return Version{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	version, ok := m.version(organisationID, versionID)
	if !ok {
		return Version{}, ErrNotFound
	}
	if version.Status != StatusDraft || version.Purpose != "" || len(version.AI.Draft) != 0 || len(version.Skills) != 0 {
		return Version{}, ErrImmutable
	}
	assignments := make([]SkillAssignment, 0, len(draft.Skills))
	seen := map[string]struct{}{}
	for _, suggested := range draft.Skills {
		if _, ok := seen[suggested.SkillID]; ok {
			return Version{}, ErrDuplicate
		}
		seen[suggested.SkillID] = struct{}{}
		skill, ok := m.skill(organisationID, suggested.SkillID)
		if !ok {
			return Version{}, ErrNotFound
		}
		if strings.TrimSpace(suggested.Definition) == "" || !validProficiency(suggested.RequiredProficiency) || !validPriority(suggested.Priority) {
			return Version{}, ErrInvalid
		}
		id, err := newID()
		if err != nil {
			return Version{}, err
		}
		assignments = append(assignments, SkillAssignment{
			ID:                  id,
			SkillID:             skill.ID,
			SkillName:           skill.Name,
			Category:            skill.Category,
			Definition:          strings.TrimSpace(suggested.Definition),
			RequiredProficiency: suggested.RequiredProficiency,
			Priority:            suggested.Priority,
			Rationale:           strings.TrimSpace(suggested.Rationale),
			Provenance:          ProvenanceAISuggested,
			Position:            len(assignments) + 1,
		})
	}
	snapshot, err := snapshotDraft(draft, assignments)
	if err != nil {
		return Version{}, err
	}
	version.Purpose = strings.TrimSpace(draft.Purpose)
	version.Skills = assignments
	generated := draft.GeneratedAt.UTC().Truncate(time.Second)
	version.AI = AIMetadata{
		Model:                strings.TrimSpace(draft.Model),
		SourceContextVersion: strings.TrimSpace(draft.SourceContextVersion),
		GeneratedAt:          &generated,
		Draft:                snapshot,
	}
	version.Revision++
	if err := m.audit(organisationID, version.ProfileID, version.ID, actorID, ActionAIDraft, clock()); err != nil {
		return Version{}, err
	}
	return cloneVersion(*version), nil
}

func (m *Memory) editable(organisationID, versionID string, expected int) (*Version, error) {
	return m.guard(organisationID, versionID, expected, StatusDraft)
}

func (m *Memory) guard(organisationID, versionID string, expected int, status string) (*Version, error) {
	version, ok := m.version(organisationID, versionID)
	if !ok {
		return nil, ErrNotFound
	}
	if version.Status != status {
		return nil, ErrImmutable
	}
	if version.Revision != expected {
		return nil, ErrConflict
	}
	return version, nil
}

func (m *Memory) current(organisationID, profileID string) (Profile, error) {
	profile, ok := m.identity(organisationID, profileID)
	if !ok {
		return Profile{}, ErrNotFound
	}
	var latest *Version
	for i := range m.versions {
		version := &m.versions[i]
		if version.ProfileID == profileID && version.OrganisationID == organisationID && (latest == nil || version.VersionNumber > latest.VersionNumber) {
			latest = version
		}
	}
	if latest == nil {
		return Profile{}, ErrNotFound
	}
	return m.profileView(profile, *latest), nil
}

func (m *Memory) profileView(profile Profile, version Version) Profile {
	profile.Version = cloneVersion(version)
	return profile
}

func (m *Memory) identity(organisationID, profileID string) (Profile, bool) {
	for _, profile := range m.profiles {
		if profile.ID == profileID && profile.OrganisationID == organisationID {
			return profile, true
		}
	}
	return Profile{}, false
}

func (m *Memory) version(organisationID, versionID string) (*Version, bool) {
	for i := range m.versions {
		if m.versions[i].ID == versionID && m.versions[i].OrganisationID == organisationID {
			return &m.versions[i], true
		}
	}
	return nil, false
}

func (m *Memory) role(organisationID, roleID string) (Role, bool) {
	for _, role := range m.roles {
		if role.ID == roleID && role.OrganisationID == organisationID {
			return role, true
		}
	}
	return Role{}, false
}

func (m *Memory) skill(organisationID, skillID string) (Skill, bool) {
	for _, skill := range m.skills {
		if skill.ID == skillID && skill.OrganisationID == organisationID {
			return skill, true
		}
	}
	return Skill{}, false
}

func (m *Memory) sources(inputs []Source) ([]Source, error) {
	sources := make([]Source, 0, len(inputs))
	for _, input := range inputs {
		id, err := newID()
		if err != nil {
			return nil, err
		}
		sources = append(sources, Source{ID: id, Title: strings.TrimSpace(input.Title), ReferenceURI: strings.TrimSpace(input.ReferenceURI)})
	}
	return sources, nil
}

func (m *Memory) audit(organisationID, profileID, versionID, actorID, action string, occurredAt time.Time) error {
	id, err := newID()
	if err != nil {
		return err
	}
	m.events = append(m.events, AuditEvent{ID: id, ProfileVersionID: versionID, ActorPersonID: actorID, Action: action, OccurredAt: occurredAt})
	_ = organisationID
	_ = profileID
	return nil
}

func (m *Memory) eventProfile(event AuditEvent) string {
	version, ok := m.versionByID(event.ProfileVersionID)
	if !ok {
		return ""
	}
	return version.ProfileID
}

func (m *Memory) versionByID(versionID string) (*Version, bool) {
	for i := range m.versions {
		if m.versions[i].ID == versionID {
			return &m.versions[i], true
		}
	}
	return nil, false
}

func (m *Memory) maxNumber(profileID string) int {
	max := 0
	for _, version := range m.versions {
		if version.ProfileID == profileID && version.VersionNumber > max {
			max = version.VersionNumber
		}
	}
	return max
}

func skillIndex(skills []SkillAssignment, id string) int {
	for i, skill := range skills {
		if skill.ID == id {
			return i
		}
	}
	return -1
}

func copyInt(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func clock() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

func cloneVersion(version Version) Version {
	cloned := version
	cloned.Skills = append([]SkillAssignment{}, version.Skills...)
	cloned.Sources = append([]Source{}, version.Sources...)
	if cloned.Skills == nil {
		cloned.Skills = []SkillAssignment{}
	}
	if cloned.Sources == nil {
		cloned.Sources = []Source{}
	}
	cloned.TeamSize = copyInt(version.TeamSize)
	if version.AI.GeneratedAt != nil {
		generated := *version.AI.GeneratedAt
		cloned.AI.GeneratedAt = &generated
	}
	if version.AI.Draft != nil {
		cloned.AI.Draft = append(json.RawMessage{}, version.AI.Draft...)
	}
	return cloned
}

func validAIDraft(draft AIDraft) error {
	if strings.TrimSpace(draft.Model) == "" || strings.TrimSpace(draft.SourceContextVersion) == "" || strings.TrimSpace(draft.Purpose) == "" || draft.GeneratedAt.IsZero() || len(draft.Skills) == 0 {
		return ErrInvalid
	}
	return nil
}

func snapshotDraft(draft AIDraft, skills []SkillAssignment) (json.RawMessage, error) {
	type skill struct {
		SkillID             string `json:"skillId"`
		Category            string `json:"category"`
		Definition          string `json:"definition"`
		RequiredProficiency int    `json:"requiredProficiency"`
		Priority            string `json:"priority"`
		Rationale           string `json:"rationale"`
	}
	body := struct {
		Model                string  `json:"model"`
		SourceContextVersion string  `json:"sourceContextVersion"`
		Purpose              string  `json:"purpose"`
		Skills               []skill `json:"skills"`
	}{
		Model:                strings.TrimSpace(draft.Model),
		SourceContextVersion: strings.TrimSpace(draft.SourceContextVersion),
		Purpose:              strings.TrimSpace(draft.Purpose),
		Skills:               make([]skill, 0, len(skills)),
	}
	for _, item := range skills {
		body.Skills = append(body.Skills, skill{
			SkillID:             item.SkillID,
			Category:            item.Category,
			Definition:          item.Definition,
			RequiredProficiency: item.RequiredProficiency,
			Priority:            item.Priority,
			Rationale:           item.Rationale,
		})
	}
	return json.Marshal(body)
}
