package pathway

import (
	"net/url"
	"strings"
)

func learnLink(baseURL, externalCourseID, planItemID string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" || externalCourseID == "" || planItemID == "" {
		return "", ErrInvalid
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", ErrInvalid
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/courses/" + url.PathEscape(externalCourseID)
	query := parsed.Query()
	query.Set("planItemId", planItemID)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
