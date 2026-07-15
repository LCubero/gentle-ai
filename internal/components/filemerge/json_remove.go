package filemerge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// RemoveTopLevelJSONKey removes root keys without rewriting unrelated bytes.
func RemoveTopLevelJSONKey(raw []byte, key string) ([]byte, bool, error) {
	if err := validateJSONCObject(raw); err != nil {
		return nil, false, fmt.Errorf("unmarshal json object: %w", err)
	}
	if i := skipJSONTrivia(raw, 0); i >= len(raw) || raw[i] != '{' {
		return nil, false, fmt.Errorf("json root is not an object")
	}
	for updated := raw; ; {
		next, removed := removeTopLevelJSONKeyOnce(updated, []byte(strconv.Quote(key)))
		if !removed {
			if err := validateJSONCObject(updated); err != nil {
				return nil, false, fmt.Errorf("validate updated json object: %w", err)
			}
			return updated, !bytes.Equal(updated, raw), nil
		}
		updated = next
	}
}

func validateJSONCObject(raw []byte) error {
	normalized := bytes.Clone(raw)
	for i := 0; i < len(raw); i++ {
		if raw[i] == '"' {
			i = scanJSONString(raw, i) - 1
			continue
		}
		if i+1 >= len(raw) || raw[i] != '/' || (raw[i+1] != '/' && raw[i+1] != '*') {
			continue
		}
		end := skipJSONComment(raw, i)
		if end > len(raw) {
			return fmt.Errorf("unterminated block comment")
		}
		for j := i; j < end; j++ {
			if raw[j] != '\n' && raw[j] != '\r' {
				normalized[j] = ' '
			}
		}
		i = end - 1
	}

	return json.Unmarshal(stripTrailingCommas(normalized), new(map[string]any))
}

func removeTopLevelJSONKeyOnce(raw, quotedKey []byte) ([]byte, bool) {
	i, previousComma := skipJSONTrivia(raw, 0)+1, -1
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
			trivia := raw[memberStart:keyStart]
			preserveComment := previousComma < 0 && bytes.IndexByte(trivia, '/') >= 0
			if preserveComment {
				commentEnd := 0
				for offset := 0; offset < len(trivia); offset++ {
					if trivia[offset] != '/' {
						continue
					}
					commentEnd = skipJSONComment(trivia, offset)
					offset = commentEnd - 1
					if commentEnd < len(trivia) && trivia[commentEnd] == '\n' {
						commentEnd++
						offset++
					}
				}
				removeStart = memberStart + commentEnd
			}
			if raw[delimiter] == ',' {
				removeEnd++
			} else if previousComma >= 0 {
				removeStart = previousComma
			}
			if preserveComment {
				if bytes.HasPrefix(raw[removeEnd:], []byte("\r\n")) {
					removeEnd += 2
				} else if bytes.HasPrefix(raw[removeEnd:], []byte("\n")) {
					removeEnd++
				}
			}
			updated := bytes.Clone(raw[:removeStart])
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
	depth := 0
	for i := start; i < len(raw); i++ {
		switch raw[i] {
		case '"':
			i = scanJSONString(raw, i) - 1
		case '/':
			i = skipJSONComment(raw, i) - 1
		case '{', '[':
			depth++
		case ']', '}':
			depth--
			if depth < 0 {
				return i
			}
		case ',':
			if depth == 0 {
				return i
			}
		}
	}
	return len(raw) - 1
}
func skipJSONTrivia(raw []byte, start int) int {
	for start < len(raw) {
		switch raw[start] {
		case ' ', '\t', '\r', '\n':
			start++
		case '/':
			start = skipJSONComment(raw, start)
		default:
			return start
		}
	}
	return start
}

func skipJSONComment(raw []byte, start int) int {
	if start+1 >= len(raw) || raw[start] != '/' {
		return start
	}
	if raw[start+1] == '/' {
		if end := bytes.IndexByte(raw[start+2:], '\n'); end >= 0 {
			return start + 2 + end
		}
		return len(raw)
	}
	if raw[start+1] == '*' {
		if end := bytes.Index(raw[start+2:], []byte("*/")); end >= 0 {
			return start + 4 + end
		}
		return len(raw) + 1
	}
	return start
}
