package composer

import (
	"errors"
	"regexp"
	"strings"
)

// This file is a go implementation of Composers SemVar VersionParser here:
// https://github.com/composer/semver/blob/2b303e43d14d15cc90c8e8db4a1cdb6259f1a5c5/src/VersionParser.php

// Copyright (C) 2015 Composer
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is furnished to do
// so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Regex to match pre-release data (sort of).
//
// Due to backwards compatibility:
//   - Instead of enforcing hyphen, an underscore, dot or nothing at all are also accepted.
//   - Only stabilities as recognized by Composer are allowed to precede a numerical identifier.
//   - Numerical-only pre-release identifiers are not supported, see tests.
//
//                        |--------------|
// [major].[minor].[patch] -[pre-release] +[build-metadata]
const ModifierRegex = `[._-]?(?:(stable|beta|b|RC|alpha|a|patch|pl|p)((?:[.-]?\d+)*)?)?([.-]?dev)?`

func ParseNumericAliasPrefix(branch string) string {
	exp := regexp.MustCompile(`(?i)^(?P<version>(\d+\.)*\d+)(?:\.x)?-dev$`)

	if r := exp.FindStringSubmatch(branch); len(r) > 0 {
		return r[1] + "."
	}

	return ""
}

func NormaliseVersion(version, fullVersion string) (string, error) {
	var index int
	version = strings.Trim(version, " ")

	if fullVersion == "" {
		fullVersion = version
	}

	// strip off aliasing
	exp := regexp.MustCompile(`^([^,\s]+) +as +([^,\s]+)$`)
	if r := exp.FindStringSubmatch(version); len(r) > 0 {
		version = r[1]
	}

	// match master-like branches
	if r, err := regexp.MatchString(`(?i)^(?:dev-)?(?:master|trunk|default)$`, version); err == nil && r {
		return "9999999-dev", nil
	} else if err != nil {
		return "", err
	}

	// if requirement is branch-like, use full name
	if len(version) > 4 && "dev-" == strings.ToLower(version[0:4]) {
		return "dev-" + version[4:], nil
	}

	// strip off build metadata
	exp = regexp.MustCompile(`^([^,\s+]+)\+[^\s]+$`)
	if r := exp.FindStringSubmatch(version); len(r) > 0 {
		version = r[1]
	}

	// match classical versioning
	exp = regexp.MustCompile(`(?i)^v?(\d{1,5})(\.\d+)?(\.\d+)?(\.\d+)?` + ModifierRegex + `$`)
	// match date(time) based versioning
	exp2 := regexp.MustCompile(`(?i)^v?(\d{4}(?:[.:-]?\d{2}){1,6}(?:[.:-]?\d{1,3})?)` + ModifierRegex + `$`)
	var r []string

	if r = exp.FindStringSubmatch(version); len(r) > 0 {
		version = r[1]

		if len(r) > 2 && r[2] != "" {
			version += r[2]
		} else {
			version += ".0"
		}

		if len(r) > 3 && r[3] != "" {
			version += r[3]
		} else {
			version += ".0"
		}

		if len(r) > 4 && r[4] != "" {
			version += r[4]
		} else {
			version += ".0"
		}

		index = 5
	} else if r = exp2.FindStringSubmatch(version); len(r) > 0 {
		exp = regexp.MustCompile(`\D`)
		version = exp.ReplaceAllString(r[1], ".")
		index = 2
	}

	// add version modifiers if a version was matched
	if index > 0 {
		if len(r) > index && r[index] != "" {
			if r[index] == "stable" {
				return version, nil
			}

			version += "-" + expandStability(r[index])

			if len(r) > index+1 && r[index+1] != "" {
				version += strings.TrimLeft(r[index+1], ".-")
			}
		}

		if len(r) > index+2 && r[index+2] != "" {
			version += "-dev"
		}

		return version, nil
	}

	// match dev branches
	exp = regexp.MustCompile(`(?i)(.*?)[.-]?dev$`)
	if r := exp.FindStringSubmatch(version); len(r) > 0 {
		return NormalizeBranch(r[1])
	}

	errMsg := ""

	if r, err := regexp.MatchString(" +as +"+regexp.QuoteMeta(version)+"$", version); err == nil && r {
		errMsg = " in \"" + fullVersion + "\", the alias must be an exact version"
	} else if err != nil {
		return "", err
	} else if r, err := regexp.MatchString("^"+regexp.QuoteMeta(version)+" +as +", version); err == nil && r {
		errMsg = " in \"" + fullVersion + "\", the alias source must be an exact version, if it is a branch name you should prefix it with dev-"
	} else if err != nil {
		return "", err
	}

	return "", errors.New("Invalid version string \"" + fullVersion + "\"" + errMsg)
}

func NormalizeBranch(name string) (string, error) {
	name = strings.Trim(name, " ")

	for _, test := range []string{"master", "trunk", "default"} {
		if name == test {
			return NormaliseVersion(name, "")
		}
	}

	exp := regexp.MustCompile(`(?i)^v?(\d+)(\.(?:\d+|[xX*]))?(\.(?:\d+|[xX*]))?(\.(?:\d+|[xX*]))?$`)

	if r := exp.FindStringSubmatch(name); len(r) > 0 {
		version := ""
		for i := 1; i < 5; i++ {
			if len(r) <= i || r[i] == "" {
				version += ".x"
				continue
			}

			part := r[i]
			part = strings.Replace(part, "*", "x", -1)
			part = strings.Replace(part, "X", "x", -1)

			version += part
		}

		return strings.Replace(version, "x", "9999999", -1) + "-dev", nil
	}

	return "dev-" + name, nil
}

func expandStability(stability string) string {
	stability = strings.ToLower(stability)

	switch stability {
	case "a":
		stability = "alpha"
	case "b":
		stability = "beta"
	case "p":
		fallthrough
	case "pl":
		stability = "patch"
	case "rc":
		stability = "RC"
	}

	return stability
}
