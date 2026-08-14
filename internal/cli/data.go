package cli

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"
)

// dataRe matches embedded {{/* __DATA__ ... */}} blocks in template content. Compiled once at package init because
// splitTemplateData runs per preload file plus once for STDIN.
// Go's regexp package shamefully doesn't support the (?x) free-spacing flag, so we assemble the pattern from string
// concatenation
var dataRe = regexp.MustCompile(
	`\{\{/\*` + // match opening comment: {{/*
		`\s*` + // optional whitespace
		`__DATA__` + // match the expected data marker
		`\s*` + // optional whitespace
		`([\s\S]*?)` + // group 1: Capture the actual data (non-greedy)
		`\*/\}\}`, // match closing comment: */}}
)

func processTemplate(template string, preloadDataBlocks []string, dataFiles []string) (string, map[string]any, error) {
	tmplContent, embeddedDataBlocks := splitTemplateData(template)

	var allDataMaps []map[string]any
	var err error

	if allDataMaps, err = collectDataFromEmbeddedBlocks(allDataMaps, preloadDataBlocks); err != nil {
		return "", nil, err
	}

	if allDataMaps, err = collectDataFromEmbeddedBlocks(allDataMaps, embeddedDataBlocks); err != nil {
		return "", nil, err
	}

	if allDataMaps, err = collectDataFromFiles(allDataMaps, dataFiles); err != nil {
		return "", nil, err
	}

	var data map[string]any
	if data, err = mergeAllDataMaps(allDataMaps); err != nil {
		return "", nil, err
	}

	// If no data provided, use empty map (allows self-contained templates using only sprig functions)
	if data == nil {
		data = make(map[string]any)
	}

	return tmplContent, data, nil
}

func collectDataFromEmbeddedBlocks(allDataMaps []map[string]any, embeddedDataBlocks []string) ([]map[string]any, error) {
	if v := os.Getenv(ENV_IGNORE_EMBED); v != "" && v != "0" && v != "false" {
		return allDataMaps, nil
	}

	for i, block := range embeddedDataBlocks {
		var blockData map[string]any
		if err := yaml.Unmarshal([]byte(block), &blockData); err != nil {
			return nil, fmt.Errorf("error parsing embedded YAML data block %d: %w", i+1, err)
		}
		allDataMaps = append(allDataMaps, blockData)
	}

	return allDataMaps, nil
}

func collectDataFromFiles(allDataMaps []map[string]any, dataFiles []string) ([]map[string]any, error) {
	for _, dataFile := range dataFiles {
		dataBytes, err := os.ReadFile(dataFile)
		if err != nil {
			return nil, fmt.Errorf("error reading data file %s: %w", dataFile, err)
		}

		var fileData map[string]any
		if err := yaml.Unmarshal(dataBytes, &fileData); err != nil {
			return nil, fmt.Errorf("error parsing YAML data from %s: %w", dataFile, err)
		}
		allDataMaps = append(allDataMaps, fileData)
	}

	return allDataMaps, nil
}

func mergeAllDataMaps(allDataMaps []map[string]any) (map[string]any, error) {
	var data map[string]any
	for _, dataMap := range allDataMaps {
		if data == nil {
			data = dataMap
			continue
		}
		if err := mergo.Merge(&data, dataMap, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("error merging data: %w", err)
		}
	}
	return data, nil
}

// splits template content from embedded data sections Returns (tmplText, dataBlocks) where dataBlocks contains all
// embedded YAML blocks
// The template text is returned unchanged since {{/* __DATA__ */}} comments are already ignored by the template parser,
// and removing them would break line number reporting in errors
func splitTemplateData(content string) (string, []string) {
	var dataBlocks []string
	for _, match := range dataRe.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			dataBlocks = append(dataBlocks, strings.TrimSpace(match[1]))
		}
	}
	return content, dataBlocks
}
