package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
)

func absPath(path string) (string, error) {
	usr, _ := user.Current()
	dir := usr.HomeDir

	if path[:2] == "~/" {
		path = strings.Replace(path, "~", dir, 1)
	}

	return filepath.Abs(path)
}

func appendHttp(url string) string {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "http://" + url
	}
	return url
}

func assetFS() http.FileSystem {
	for k := range _bintree.Children {
		return http.Dir(k)
	}
	panic("unreachable")
}

func extractPageTitle(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	checkErr(err)

	re := regexp.MustCompile("<title>(.*?)</title>")
	result := re.FindStringSubmatch(string(body))
	if len(result) == 2 {
		return result[1], nil
	}
	return "", errors.New(
		fmt.Sprintf("<title>...</title> not found in %s url", url),
	)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func extractTags(search string) (result []string) {
	re := regexp.MustCompile("\\[(.*?)\\]")

	for _, submatch := range re.FindAllStringSubmatch(search, -1) {
		if len(submatch) > 1 {
			result = append(result, strings.TrimSpace(submatch[1]))
		}
	}

	return result
}

func removeTags(search string) string {
	re := regexp.MustCompile("(\\[.*?\\])")
	return strings.TrimSpace(re.ReplaceAllString(search, ""))
}
