package main

import (
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	flags "github.com/jessevdk/go-flags"
)

type Options struct {
	Count bool `short:"c" long:"count" description:"Only count listed HTML files (Disabled by default)"`
	Date  bool `short:"d" long:"date" description:"Print last modified date (Disabled by default)"`
	Fetch bool `short:"f" long:"fetch" description:"Fetch listed HTML files (Disabled by default)"`
	Wait  int  `short:"w" long:"wait" default:"200" description:"Fetch duration (by milli seconds)"`
}

var opts Options

type Sitemap struct {
	Page []struct {
		Url     string    `xml:"loc"`
		Lastmod time.Time `xml:"lastmod"`
	} `xml:"url"`
}

func main() {
	start := time.Now()

	parser := flags.NewParser(&opts, flags.Default)
	parser.Name = "gethtmlmapped"
	parser.Usage = "http://example.com/sitemap.xml [OPTIONS]"

	args, err := parser.Parse()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if len(args) == 0 {
		parser.WriteHelp(os.Stdout)
		os.Exit(1)
	}

	sitemap, err := httpGet(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	contents := Sitemap{}
	if err := xml.Unmarshal([]byte(sitemap), &contents); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	repl := regexp.MustCompile(`^https?://`)

	topdir := strings.Split(repl.ReplaceAllString(contents.Page[0].Url, ""), "/")[0]
	if _, err := os.Stat(topdir); opts.Fetch && err == nil {
		fmt.Fprintln(os.Stderr, topdir+" directory is already exists")
		os.Exit(1)
	}

	wait := time.Duration(opts.Wait) * time.Millisecond
	succeed := []string{}
	failed := []string{}
	latest, _ := time.Parse(time.RFC3339, time.RFC3339)
	for _, page := range contents.Page {
		if page.Lastmod.Unix() > latest.Unix() {
			latest = page.Lastmod
		}

		if !opts.Count {
			info := page.Url
			if opts.Date {
				info = info + "\t" + page.Lastmod.Format("2006-01-02 15:04:05")
			}
			fmt.Println(info)
		}

		if !opts.Fetch {
			continue
		}

		filepath := repl.ReplaceAllString(page.Url, "")
		dir, _ := path.Split(filepath)

		content, err := httpGet(page.Url)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed = append(failed, page.Url)
			continue
		}

		if _, err := os.Stat(dir); err != nil {
			if err := os.MkdirAll(dir, 0777); err != nil {
				fmt.Fprintln(os.Stderr, err)
				failed = append(failed, page.Url)
				continue
			}
		}

		fpw, err := os.Create(filepath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed = append(failed, page.Url)
			continue
		}

		w := bufio.NewWriter(fpw)
		fmt.Fprint(w, content)
		w.Flush()
		fpw.Close()

		succeed = append(succeed, page.Url)
		time.Sleep(wait)
	}

	if opts.Count {
		info := strconv.Itoa(len(contents.Page))
		if opts.Date {
			info = info + "\t" + latest.Format("2006-01-02 15:04:05")
		}
		fmt.Println(info)
		os.Exit(0)
	}

	if opts.Fetch {
		end := time.Now()
		fmt.Println("succeed : " + string(len(succeed)))
		fmt.Println("failed : " + string(len(failed)))
		fmt.Printf("%f sec.\n", (end.Sub(start)).Seconds())
		os.Exit(0)
	}
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
