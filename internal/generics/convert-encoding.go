package generics

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

func urlToText(input string) (*ConvertResult, error) {
	decoded, err := url.QueryUnescape(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decode URL: %w", err)
	}
	return &ConvertResult{Output: decoded}, nil
}

func textToUrl(input string) (*ConvertResult, error) {
	return &ConvertResult{Output: url.QueryEscape(input)}, nil
}

func jwtDecode(tokenString string) (*ConvertResult, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format: expected 3 parts separated by '.'")
	}
	header, err := jwtDecodeSegment(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode header: %w", err)
	}
	payload, err := jwtDecodeSegment(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	return &ConvertResult{
		Tables: []TableData{
			{Headers: []string{"Header", "Value"}, Rows: dictToRows(header)},
			{Headers: []string{"Payload", "Value"}, Rows: dictToRows(payload)},
		},
	}, nil
}

func dictToRows(dict map[string]any) [][]string {
	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	rows := make([][]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, []string{k, formatJWTValue(dict[k])})
	}
	return rows
}

func formatJWTValue(v any) string {
	switch v := v.(type) {
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func jwtDecodeSegment(seg string) (map[string]any, error) {
	if l := len(seg) % 4; l > 0 {
		seg += strings.Repeat("=", 4-l)
	}
	decoded, err := base64.URLEncoding.DecodeString(seg)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(decoded, &result); err != nil {
		return nil, err
	}
	return result, nil
}

