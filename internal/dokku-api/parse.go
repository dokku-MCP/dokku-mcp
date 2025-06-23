package dokkuApi

import (
	"strings"
)

// ParseKeyValueOutput parses key-value output (e.g., key: value or key=value) from Dokku CLI.
func ParseKeyValueOutput(output string, separator string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, separator) {
			parts := strings.SplitN(line, separator, 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if key != "" {
					result[key] = value
				}
			}
		}
	}

	return result
}

// ParseListOutput parses a list output (one item per line, optionally skipping headers/empty lines).
func ParseListOutput(output string, filterEmpty bool) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip headers and empty lines if requested
		if filterEmpty && (line == "" || strings.HasPrefix(line, "====") || strings.Contains(line, "NAME")) {
			continue
		}

		// For service lists, take the first column (service name)
		if strings.Contains(line, " ") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				result = append(result, parts[0])
			}
		} else if line != "" {
			result = append(result, line)
		}
	}

	return result
}

// ParseTableOutput parses table output (first line is header, rest are rows).
func ParseTableOutput(output string, skipHeaders bool) []map[string]string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return nil
	}

	var headerLine string
	var dataLines []string

	if skipHeaders {
		// Find the header line (usually contains column names)
		for i, line := range lines {
			if strings.Contains(line, "NAME") || strings.Contains(line, "STATUS") {
				headerLine = line
				dataLines = lines[i+1:]
				break
			}
		}
		if headerLine == "" && len(lines) > 1 {
			headerLine = lines[0]
			dataLines = lines[1:]
		}
	} else {
		headerLine = lines[0]
		dataLines = lines[1:]
	}

	if headerLine == "" {
		return nil
	}

	headers := strings.Fields(headerLine)
	var result []map[string]string

	for _, line := range dataLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "====") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		row := make(map[string]string)
		for i, header := range headers {
			if i < len(fields) {
				row[header] = fields[i]
			} else {
				row[header] = ""
			}
		}
		result = append(result, row)
	}

	return result
}

// ParseColonKeyValueLine parses a single colon-separated key-value line, trims spaces, and returns key, value, and ok.
func ParseColonKeyValueLine(line string) (string, string, bool) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	return key, value, true
}

// ParseLinesSkipHeaders parses output into lines, skipping common Dokku headers and empty lines.
func ParseLinesSkipHeaders(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and common Dokku headers
		if line == "" || strings.HasPrefix(line, "====") || strings.Contains(line, "NAME") {
			continue
		}
		result = append(result, line)
	}

	return result
}

// ParseFieldsOutput parses output where each line contains space-separated fields.
// Returns a slice of field slices for each line.
func ParseFieldsOutput(output string, skipHeaders bool) [][]string {
	lines := ParseLinesSkipHeaders(output)
	var result [][]string

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			result = append(result, fields)
		}
	}

	return result
}

// ParseTrimmedLines parses output into trimmed lines, optionally filtering empty lines.
func ParseTrimmedLines(output string, filterEmpty bool) []string {
	lines := strings.Split(output, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !filterEmpty || line != "" {
			result = append(result, line)
		}
	}

	return result
}
