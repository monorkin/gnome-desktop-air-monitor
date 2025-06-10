package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const OUTPUT_FILE_PATH = "THIRD_PARTY_LICENSES"

type Module struct {
	Path    string
	Version string
	Dir     string
	Replace *Module
}

type LicenseEntry struct {
	Module string
	URL    string
	Type   string // either local path or URL
	Text   string
}

func main() {
	println("Bundling third-party licenses")

	println("Fetching licenses of dependencies...")
	cmd := exec.Command("go-licenses", "csv", "./...")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ Failed to run go-licenses csv: %v\n", err)
		os.Exit(1)
	}

	println("Parsing result...")

	var entries []LicenseEntry
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}
		module, url, licenseType := parts[0], parts[1], parts[2]
		text, err := downloadText(url)
		if err != nil {
			fmt.Printf("❌  Failed to download license for %s: %v", module, err)
			continue
		}
		text = sanitizeText(text)
		fmt.Printf("✅  Downloaded license text for module: %s\n", module)
		entries = append(entries, LicenseEntry{
			Module: module,
			URL:    url,
			Type:   licenseType,
			Text:   text,
		})
	}

	println("Writing license bundle...")
	writeCombinedLicense(entries)

	fmt.Printf("Done! License bundle written to %s", OUTPUT_FILE_PATH)
}

func downloadText(url string) (string, error) {
	url = sanitizeURL(url)

	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to download %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return string(body), err
}

func sanitizeURL(url string) string {
	if strings.HasPrefix(url, "https://github.com/") {
		url = strings.Replace(url, "https://github.com/", "https://raw.githubusercontent.com/", 1)
		url = strings.Replace(url, "/blob/", "/", 1)
	} else if strings.HasPrefix(url, "https://cs.opensource.google/go/x/") {
		url = strings.Replace(url, "https://cs.opensource.google/go/x/", "raw.githubusercontent.com/golang/", 1)
		url = strings.Replace(url, "/+/", "/", 1)
		url = strings.Replace(url, ":", "/", 1)
		url = "https://" + url
	}

	return url
}

func writeCombinedLicense(entries []LicenseEntry) {
	out, err := os.Create(OUTPUT_FILE_PATH)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	fmt.Fprintln(out, "THIRD PARTY LICENSES")
	fmt.Fprintf(out, "Generated on: %s\n", now)

	for _, e := range entries {
		library := fmt.Sprintf("LIBRARY: %s", e.Module)
		license := fmt.Sprintf("LICENSE: %s", e.Type)
		url := fmt.Sprintf("URL: %s", e.URL)

		widest := max(len(library), len(license), len(url))

		border := "+-" + strings.Repeat("-", widest) + "-+"
		libraryLine := fmt.Sprintf("| %-*s |", widest, library)
		licenseLine := fmt.Sprintf("| %-*s |", widest, license)
		urlLine := fmt.Sprintf("| %-*s |", widest, url)
		header := strings.Join([]string{
			border,
			libraryLine,
			licenseLine,
			urlLine,
			border,
		}, "\n")

		fmt.Fprintln(out, "")
		fmt.Fprintln(out, header)
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, e.Text)
	}
}

func sanitizeText(text string) string {
	if strings.Contains(text, "<html") {
		doc, err := html.Parse(strings.NewReader(text))
		if err != nil {
			fmt.Printf("❌  Failed to parse HTML: %v\n", err)
			return text
		}

		var body *html.Node
		var findBody func(*html.Node)
		findBody = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "body" {
				body = n
				return
			}
			for c := n.FirstChild; c != nil && body == nil; c = c.NextSibling {
				findBody(c)
			}
		}
		findBody(doc)

		if body == nil {
			body = doc
		}

		var b strings.Builder
		var extract func(*html.Node)
		extract = func(n *html.Node) {
			if n.Type == html.TextNode {
				b.WriteString(n.Data)
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				extract(c)
			}
		}
		extract(body)

		return strings.TrimSpace(b.String())
	}

	return strings.TrimSpace(text)
}
