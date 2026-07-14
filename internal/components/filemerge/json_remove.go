package filemerge

import (
	"bytes"
	"fmt"
	"strconv"
)

// RemoveTopLevelJSONKey removes root keys without rewriting unrelated bytes.
func RemoveTopLevelJSONKey(raw []byte, key string) ([]byte, bool, error) {
	if _, err := UnmarshalJSONObject(raw); err != nil {
		return nil, false, fmt.Errorf("unmarshal json object: %w", err)
	}
	i := skipJSONTrivia(raw, 0)
	if i >= len(raw) || raw[i] != '{' {
		return nil, false, fmt.Errorf("json root is not an object")
	}
	updated := raw
	changed := false
	for {
		next, removed := removeTopLevelJSONKeyOnce(updated, []byte(strconv.Quote(key)))
		if !removed {
			return updated, changed, nil
		}
		updated = next
		changed = true
	}
}

func removeTopLevelJSONKeyOnce(raw, quotedKey []byte) ([]byte, bool) {
	i := skipJSONTrivia(raw, 0) + 1
	previousComma := -1
	for {
		memberStart := i
		i = skipJSONTrivia(raw, i)
		if i >= len(raw) || raw[i] == '}' {
			return raw, false
		}
		keyStart := i
		keyEnd := scanJSONString(raw, keyStart)
		i = skipJSONTrivia(raw, keyEnd) + 1 // validated input guarantees the colon
		delimiter := scanJSONValueDelimiter(raw, skipJSONTrivia(raw, i))
		if bytes.Equal(raw[keyStart:keyEnd], quotedKey) {
			removeStart, removeEnd := memberStart, delimiter
			if raw[delimiter] == ',' {
				removeEnd++
			} else if previousComma >= 0 {
				removeStart = previousComma
			}
			updated := make([]byte, 0, len(raw)-(removeEnd-removeStart))
			updated = append(updated, raw[:removeStart]...)
			updated = append(updated, raw[removeEnd:]...)
			return updated, true
		}
		if raw[delimiter] == '}' {
			return raw, false
		}
		previousComma, i = delimiter, delimiter+1
	}
}
func scanJSONString(raw []byte, start int) int {
	escaped := false
	for i := start + 1; i < len(raw); i++ {
		if escaped {
			escaped = false
		} else if raw[i] == '\\' {
			escaped = true
		} else if raw[i] == '"' {
			return i + 1
		}
	}
	return len(raw)
}

func scanJSONValueDelimiter(raw []byte, start int) int {
	depth, inString, escaped := 0, false, false
	for i := start; i < len(raw); i++ {
		if inString {
			if escaped {
				escaped = false
			} else if raw[i] == '\\' {
				escaped = true
			} else if raw[i] == '"' {
				inString = false
			}
			continue
		}
		if raw[i] == '"' {
			inString = true
		} else if raw[i] == '/' {
			i = skipJSONComment(raw, i) - 1
		} else if raw[i] == '{' || raw[i] == '[' {
			depth++
		} else if raw[i] == ']' || (raw[i] == '}' && depth > 0) {
			depth--
		} else if depth == 0 && (raw[i] == ',' || raw[i] == '}') {
			return i
		}
	}
	return len(raw) - 1
}

func skipJSONTrivia(raw []byte, start int) int {
	for start < len(raw) {
		if bytes.ContainsRune([]byte(" \t\r\n"), rune(raw[start])) {
			start++
		} else if raw[start] == '/' {
			start = skipJSONComment(raw, start)
		} else {
			break
		}
	}
	return start
}

func skipJSONComment(raw []byte, start int) int {
	if start+1 >= len(raw) || raw[start] != '/' {
		return start
	}
	if raw[start+1] == '/' {
		start += 2
		for start < len(raw) && raw[start] != '\n' {
			start++
		}
		return start
	}
	if raw[start+1] == '*' {
		start += 2
		for start+1 < len(raw) && !(raw[start] == '*' && raw[start+1] == '/') {
			start++
		}
		return start + 2
	}
	return start
}
