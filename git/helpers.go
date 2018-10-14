package git

import (
	"errors"
	"fmt"
	url2 "net/url"
	"strings"
)

func GitUrlToDirectory(url string) (string, error) {
	// "ssh://" is required otherwise go will refuse to parse the string
	// also it doesn't need to be ssh:// :)
	urlInfo, err := url2.Parse("ssh://" + url)

	if err != nil {
		return "", errors.New("Unable to parse url " + url)
	}

	host := strings.Replace(urlInfo.Host, ".", "_", -1)
	host = strings.Replace(host, ":", "_", -1)

	path := strings.Replace(urlInfo.Path, "/", "", -1)
	path = strings.Replace(path, ".git", "", -1)

	return strings.ToLower(fmt.Sprintf("%s_%s", host, path)), nil
}
