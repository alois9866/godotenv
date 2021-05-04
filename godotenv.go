// Package godotenv is a rethinking of the library https://github.com/joho/godotenv
//
// Examples/readme can be found on the github page at https://github.com/alois9866/godotenv
//
// The TL;DR is that you make a .env file that looks something like:
//
// 		SOME_ENV_VAR=somevalue
//
// and then in your go code you can call:
//
// 		godotenv.Read()
//
// TODO
package godotenv

import (
	"bufio"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	singleQuotesRegex  = regexp.MustCompile(`\A'(.*)'\z`)
	doubleQuotesRegex  = regexp.MustCompile(`\A"(.*)"\z`)
	escapeRegex        = regexp.MustCompile(`\\.`)
	unescapeCharsRegex = regexp.MustCompile(`\\([^$])`)
	exportRegex        = regexp.MustCompile(`^\s*(?:export\s+)?(.*?)\s*$`)
	expandVarRegex     = regexp.MustCompile(`(\\)?(\$)(\()?{?([A-Z0-9_]+)?}?`)
)

// Read will read your env file(s) and return them as a map.
//
// If you call Read without any args it will default to reading .env in the current path.
//
// You can otherwise tell it which files to read (there can be more than one) like:
//
//		godotenv.Read("fileone", "filetwo")
//
func Read(filenames ...string) (map[string]string, error) {
	envMap := make(map[string]string)

	for _, filename := range filenamesOrDefault(filenames) {
		individualEnvMap, individualErr := readFile(filename)
		if individualErr != nil {
			return envMap, individualErr
		}

		for k, v := range individualEnvMap {
			envMap[k] = v
		}
	}

	return envMap, nil
}

func filenamesOrDefault(filenames []string) []string {
	if len(filenames) == 0 {
		return []string{".env"}
	}
	return filenames
}

func readFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parse(file)
}

func parse(r io.Reader) (map[string]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	err := scanner.Err()
	if err != nil {
		return nil, err
	}

	envMap := make(map[string]string)

	for _, line := range lines {
		if !isIgnoredLine(line) {
			k, v, err := parseLine(line, envMap)
			if err != nil {
				return envMap, err
			}
			envMap[k] = v
		}
	}

	return envMap, err
}

func parseLine(line string, envMap map[string]string) (key string, value string, err error) {
	if len(line) == 0 {
		return "", "", errors.New("empty string")
	}

	line = removeComments(line)

	firstEquals := strings.Index(line, "=")
	firstColon := strings.Index(line, ":")
	splitString := strings.SplitN(line, "=", 2)
	if firstColon != -1 && (firstColon < firstEquals || firstEquals == -1) {
		// This is a yaml-style line.
		splitString = strings.SplitN(line, ":", 2)
	}
	if len(splitString) != 2 {
		return "", "", errors.New("can't separate key from value")
	}

	key = exportRegex.ReplaceAllString(splitString[0], "$1")
	value = parseValue(splitString[1], envMap)

	return key, value, err
}

// Ditch the comments (but keep quoted hashes).
func removeComments(line string) string {
	if strings.Contains(line, "#") {
		quotesAreOpen := false
		var segmentsToKeep []string
		for _, segment := range strings.Split(line, "#") {
			if strings.Count(segment, `"`) == 1 || strings.Count(segment, `'`) == 1 {
				if quotesAreOpen {
					quotesAreOpen = false
					segmentsToKeep = append(segmentsToKeep, segment)
				} else {
					quotesAreOpen = true
				}
			}

			if len(segmentsToKeep) == 0 || quotesAreOpen {
				segmentsToKeep = append(segmentsToKeep, segment)
			}
		}

		line = strings.Join(segmentsToKeep, "#")
	}

	return line
}

func parseValue(value string, envMap map[string]string) string {
	value = strings.Trim(value, " ")

	// Check if we've got quoted values or possible escapes.
	if len(value) > 1 {
		singleQuotes := singleQuotesRegex.FindStringSubmatch(value)
		doubleQuotes := doubleQuotesRegex.FindStringSubmatch(value)

		if singleQuotes != nil || doubleQuotes != nil {
			// Pull the quotes off the edges.
			value = value[1 : len(value)-1]
		}

		if doubleQuotes != nil {
			// Expand newlines.
			value = escapeRegex.ReplaceAllStringFunc(value, func(match string) string {
				c := strings.TrimPrefix(match, `\`)
				switch c {
				case "n":
					return "\n"
				case "r":
					return "\r"
				default:
					return match
				}
			})
			// Unescape characters.
			value = unescapeCharsRegex.ReplaceAllString(value, "$1")
		}

		if singleQuotes == nil {
			value = expandVariables(value, envMap)
		}
	}

	return value
}

func expandVariables(str string, m map[string]string) string {
	return expandVarRegex.ReplaceAllStringFunc(str, func(s string) string {
		submatch := expandVarRegex.FindStringSubmatch(s)

		if submatch == nil {
			return s
		}
		if submatch[1] == `\` || submatch[2] == "(" {
			return submatch[0][1:]
		}
		if submatch[4] != "" {
			return m[submatch[4]]
		}

		return s
	})
}

func isIgnoredLine(line string) bool {
	trimmedLine := strings.TrimSpace(line)
	return len(trimmedLine) == 0 || strings.HasPrefix(trimmedLine, "#")
}
