package composer

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

//noinspection GoNameStartsWithPackageName
type ComposerFile map[string]interface{}

type Source struct {
	Type      string `json:"type"`
	Url       string `json:"url"`
	Reference string `json:"reference"`
}

func DeriveVersion(tag string, isBranch bool) (version string) {
	if isBranch == false {
		version = tag
		return
	}

	tag = strings.TrimPrefix(tag, "origin/")
	version = "dev-" + tag

	// If the branch name begins with an integer, we'll assume its a version branch and
	// append ".x-dev" to the end of it. Instead of "dev-" at the beginning.
	if _, err := strconv.Atoi(string(tag[0])); err == nil {
		version = tag + ".x-dev"
	}

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

func MutateComposerFile(path, version string, source *Source) error {
	data, err := LoadFile(path)

	if err != nil {
		return err
	}

	data["version"] = version

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
