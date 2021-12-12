// Package godotenv is a rethinking of the library https://github.com/joho/godotenv
//
// Examples/readme can be found on the github page at https://github.com/alois9866/godotenv
//
// The TL;DR is that you make a .env file that looks something like:
//
// 		SOME_ENV_VAR=somevalue
//
// Then if you want to read all environment variables (from both the file and the system), you can call:
//
// 		godotenv.Get()
//
// By default, dotenv variables will take precedence over system variables.
// If you want to use values from system environment over values from dotenv files, you can use this:
//
//		godotenv.Get(PrioritizeSystem())
//
// If you want to check that some specific variables are available, you can call:
//
//		godotenv.Get(Variables("ENV_VAR1", "ENV_VAR2"))
//
// If you want to use files other than .env, you can do that too:
//
//		godotenv.Get(Variables("ENV_VAR1", "ENV_VAR2"), From("file1", "file2"))
//
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

type config struct {
	variables   []string
	filenames   []string
	systemFirst bool
}

type Option func(cfg *config)

// Variables specifies the list of variables Get... functions should look for.
//
// If the list is empty, all variables will be acquired.
func Variables(variables ...string) Option {
	return func(cfg *config) {
		cfg.variables = variables
	}
}

// PrioritizeSystem orders to use variable's value from system environment, if it is present both there and in dotenv files.
func PrioritizeSystem() Option {
	return func(cfg *config) {
		cfg.systemFirst = true
	}
}

// From specifies which files or directories should be checked for environment variables.
//
// Without this option, the .env file is used by default.
func From(filePaths ...string) Option {
	return func(cfg *config) {
		cfg.filenames = filePaths
	}
}

// Get returns a map of found environment variables with their values and a list of not found variables.
//
// In order to modify its behavior, you can provide several options:
//
//		Variables option: to specify the list of variables to get.
//		Default: all variables.
//
//		PrioritizeSystem option: to choose system variable's value, if a variable is present in both system environment and dotenv files.
//		Default: dotenv overrides system.
//
//		From option: to get variables from specific files or directories.
//		Default: from .env file.
//
func Get(options ...Option) (envMap map[string]string, notFoundVariables []string) {
	cfg := config{}
	for _, op := range options {
		op(&cfg)
	}
	return get(cfg)
}

func get(cfg config) (envMap map[string]string, notFoundVariables []string) {
	inFileVariables, _ := read(filenamesOrDefault(cfg.filenames))

	if len(cfg.variables) == 0 {
		return getAllVariables(inFileVariables, cfg.systemFirst), nil
	}

	envMap = make(map[string]string)

	for _, variable := range cfg.variables {
		set := false
		value := os.Getenv(variable)
		if value != "" {
			envMap[variable] = value
			if cfg.systemFirst {
				continue
			}
			set = true
		}

		if value, ok := inFileVariables[variable]; ok {
			envMap[variable] = value
			continue
		}

		if !set {
			notFoundVariables = append(notFoundVariables, variable)
		}
	}

	return envMap, notFoundVariables
}

func read(filenames []string) (map[string]string, error) {
	envMap := make(map[string]string)

	for _, filename := range filenames {
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
	return key, value, nil
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

func getAllVariables(fromEnvDotFiles map[string]string, systemFirst bool) map[string]string {
	envMap := make(map[string]string)

	for k, v := range fromEnvDotFiles {
		envMap[k] = v
	}

	for k, v := range systemVariables() {
		if _, ok := envMap[k]; ok && systemFirst || !ok {
			envMap[k] = v
		}
	}

	return envMap
}

func systemVariables() map[string]string {
	envMap := make(map[string]string)

	for _, rawEnvLine := range os.Environ() {
		keyValue := strings.SplitN(rawEnvLine, "=", 2)
		envMap[keyValue[0]] = keyValue[1]
	}

	return envMap
}
