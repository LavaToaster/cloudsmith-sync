package composer

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

//noinspection GoNameStartsWithPackageName
type ComposerFile map[string]interface{}

type Source struct {
	Type      string `json:"type"`
	Url       string `json:"url"`
	Reference string `json:"reference"`
}

func DeriveVersion(tagOrBranchName string, isBranch bool) (version string, normalizedVersion string, error error) {
	version = tagOrBranchName

	if isBranch == false {
		// strip the release- prefix from tags if present
		version = strings.Replace(version, "release-", "", -1)
	} else {
		rawBranch := strings.Replace(version, "origin/", "", 1)
		parsedBranch, err := NormalizeBranch(rawBranch)

		if err != nil || "9999999-dev" == parsedBranch {
			version = "dev-" + rawBranch
		} else {
			prefix := ""

			if rawBranch[0:1] == "v" {
				prefix = "v"
			}

			exp := regexp.MustCompile(`(\.9{7})+`)

			version = prefix + exp.ReplaceAllString(parsedBranch, ".x")
		}
	}

	parsedVersion, err := NormaliseVersion(version, "")

	if err != nil {
		return "", "", err
	}

	normalizedVersion = parsedVersion

	return
}

func LoadFile(path string) (file ComposerFile, error error) {
	rawComposerFile, err := ioutil.ReadFile(path + "/composer.json")

	if err != nil {
		return nil, err
	}

	error = json.Unmarshal(rawComposerFile, &file)

	return
}

func MutateComposerFile(path, version, normalizedVersion string, source *Source) error {
	data, err := LoadFile(path)

	if err != nil {
		return err
	}

	data["version"] = version
	data["version_normalized"] = normalizedVersion

	if source != nil {
		data["source"] = source
	}

	// Truncate on open, and in write mode only
	file, err := os.OpenFile(path+"/composer.json", os.O_TRUNC|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}
	defer file.Close()

	// Required to prevent goland from escaping "<", ">", and "&".
	enc := json.NewEncoder(file)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")

	return enc.Encode(&data)
}
