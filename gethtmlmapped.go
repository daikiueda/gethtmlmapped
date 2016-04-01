package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type Sitemap struct {
	Page []struct {
		Url string `xml:"loc"`
	} `xml:"url"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "URL required")
		os.Exit(1)
	}

	sitemap, err := httpGet(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	contents := Sitemap{}
	err = xml.Unmarshal([]byte(sitemap), &contents)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, page := range contents.Page {
		fmt.Println(page.Url)
	}
	//fmt.Println(len(contents.Page))
}

func httpGet(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New(resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
