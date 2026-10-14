package plan

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
)

const matchPrompt = `Return JSON only. Do not use markdown.
For each skill, choose the existing rubric band that matches each answer, in answer order.
Return band levels only. Do not return descriptors or a percentage.
Shape:
{"skills":[{"requirementId":"","levels":[0]}]}`

type BandMatcher interface {
	Match(ctx context.Context, request MatchRequest) ([]byte, error)
}

type MatchRequest struct {
	Skills []matchSkill
}

type matchSkill struct {
	RequirementID string      `json:"requirementId"`
	Rubric        []matchBand `json:"rubric"`
	Answers       []string    `json:"answers"`
}

type matchBand struct {
	Level      int    `json:"level"`
	Descriptor string `json:"descriptor"`
}

type matchResponse struct {
	Skills []struct {
		RequirementID string `json:"requirementId"`
		Levels        []int  `json:"levels"`
	} `json:"skills"`
}

func propose(ctx context.Context, store Store, matcher BandMatcher, organisationID string, session Session) ([]Proposal, error) {
	if matcher == nil {
		return nil, ErrProvider
	}
	row, err := store.GetPlan(ctx, organisationID, session.PlanID)
	if err != nil {
		return nil, err
	}
	request := MatchRequest{}
	for _, skill := range row.Skills {
		if len(skill.AssessmentItems) == 0 {
			continue
		}
		answers, err := answersFor(session, skill)
		if err != nil {
			return nil, err
		}
		bands := make([]matchBand, 0, len(skill.Rubric))
		for _, band := range skill.Rubric {
			bands = append(bands, matchBand{Level: band.Level, Descriptor: band.Descriptor})
		}
		request.Skills = append(request.Skills, matchSkill{
			RequirementID: skill.RequirementID,
			Rubric:        bands,
			Answers:       answers,
		})
	}
	raw, err := matcher.Match(ctx, request)
	if err != nil {
		return nil, err
	}
	proposals, err := proposalsFromModel(row, raw)
	if err != nil {
		return nil, err
	}
	if err := acceptProposals(row, proposals); err != nil {
		return nil, err
	}
	return proposals, nil
}

func answersFor(session Session, skill PlanSkill) ([]string, error) {
	written := map[string]string{}
	for _, card := range session.Skills {
		for _, item := range card.Items {
			written[item.ID] = item.Answer
		}
	}
	answers := make([]string, 0, len(skill.AssessmentItems))
	for _, item := range skill.AssessmentItems {
		answer, ok := written[item.ID]
		if !ok || !present(answer) {
			return nil, ErrInvalid
		}
		answers = append(answers, answer)
	}
	return answers, nil
}

func proposalsFromModel(row Plan, raw []byte) ([]Proposal, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var payload matchResponse
	if err := decoder.Decode(&payload); err != nil {
		return nil, ErrInvalid
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, ErrInvalid
	}
	byRequirement := map[string][]int{}
	for _, skill := range payload.Skills {
		if _, exists := byRequirement[skill.RequirementID]; exists || !present(skill.RequirementID) {
			return nil, ErrInvalid
		}
		byRequirement[skill.RequirementID] = skill.Levels
	}
	proposals := make([]Proposal, 0)
	for _, skill := range row.Skills {
		if len(skill.AssessmentItems) == 0 {
			continue
		}
		levels, ok := byRequirement[skill.RequirementID]
		if !ok {
			return nil, ErrInvalid
		}
		delete(byRequirement, skill.RequirementID)
		source, err := evidenceSource(skill.RecommendedMethod)
		if err != nil {
			return nil, err
		}
		percentage, err := percentageFromBands(levels)
		if err != nil {
			return nil, err
		}
		proposals = append(proposals, Proposal{
			PlanSkillID:       skill.ID,
			RequirementID:     skill.RequirementID,
			CapabilityID:      skill.CapabilityID,
			RequiredLevel:     skill.RequiredLevel,
			RecommendedMethod: skill.RecommendedMethod,
			EvidenceSource:    source,
			BandLevels:        append([]int(nil), levels...),
			Percentage:        percentage,
		})
	}
	if len(byRequirement) != 0 {
		return nil, ErrInvalid
	}
	return proposals, nil
}

func acceptProposals(row Plan, proposals []Proposal) error {
	expected := 0
	for _, skill := range row.Skills {
		if len(skill.AssessmentItems) > 0 {
			expected++
		}
	}
	if len(proposals) != expected {
		return ErrInvalid
	}
	seen := map[string]Proposal{}
	for _, proposal := range proposals {
		if _, exists := seen[proposal.RequirementID]; exists {
			return ErrInvalid
		}
		seen[proposal.RequirementID] = proposal
	}
	for _, skill := range row.Skills {
		if len(skill.AssessmentItems) == 0 {
			continue
		}
		proposal, ok := seen[skill.RequirementID]
		if !ok || proposal.PlanSkillID != skill.ID || proposal.CapabilityID != skill.CapabilityID || proposal.RequiredLevel != skill.RequiredLevel || proposal.RecommendedMethod != skill.RecommendedMethod {
			return ErrInvalid
		}
		source, err := evidenceSource(skill.RecommendedMethod)
		if err != nil || proposal.EvidenceSource != source || len(proposal.BandLevels) != len(skill.AssessmentItems) {
			return ErrInvalid
		}
		allowed := map[int]struct{}{}
		for _, band := range skill.Rubric {
			allowed[band.Level] = struct{}{}
		}
		for _, level := range proposal.BandLevels {
			if _, ok := allowed[level]; !ok {
				return ErrInvalid
			}
		}
		percentage, err := percentageFromBands(proposal.BandLevels)
		if err != nil || percentage != proposal.Percentage {
			return ErrInvalid
		}
	}
	return nil
}

func evidenceSource(method string) (string, error) {
	switch method {
	case MethodKnowledgeTest:
		return "knowledge", nil
	case MethodCaseStudy:
		return "situational_judgement", nil
	case MethodAvatarRoleplay:
		return "ai_interview", nil
	default:
		return "", ErrInvalid
	}
}

func percentageFromBands(levels []int) (int, error) {
	if len(levels) == 0 {
		return 0, ErrInvalid
	}
	sum := 0
	for _, level := range levels {
		if level < 0 || level > 4 {
			return 0, ErrInvalid
		}
		sum += level
	}
	percentage := int(math.Round(float64(sum) / float64(len(levels)) / 4 * 100))
	if percentage < 0 || percentage > 100 {
		return 0, ErrInvalid
	}
	return percentage, nil
}
