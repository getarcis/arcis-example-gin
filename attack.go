// Fire 8 attack payloads at the running Gin server and report which
// ones Arcis blocks. Run after `go run .`. Expected: every attack
// returns 403, every safe payload returns 200.
//
// Build tag keeps this file out of the main package build.

//go:build attack

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

const (
	colGreen  = "\033[32m"
	colRed    = "\033[31m"
	colYellow = "\033[33m"
	colReset  = "\033[0m"
)

func base() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

type test struct {
	category string
	label    string
	send     func() (*http.Response, error)
	expected int
}

func get(path string, params url.Values) (*http.Response, error) {
	u := base() + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	return http.Get(u)
}

func post(path string, body string) (*http.Response, error) {
	return http.Post(base()+path, "application/json", bytes.NewBufferString(body))
}

func main() {
	tests := []test{
		{"safe", "safe input", func() (*http.Response, error) { return get("/api/echo", url.Values{"q": {"hello"}}) }, 200},
		{"xss", "<script> in query", func() (*http.Response, error) { return get("/api/echo", url.Values{"q": {"<script>alert(1)</script>"}}) }, 403},
		{"xss", "event handler", func() (*http.Response, error) { return post("/api/echo", `{"x":"<img onerror=\"alert(1)\">"}`) }, 403},
		{"sql", "'; DROP TABLE users; --", func() (*http.Response, error) { return get("/api/echo", url.Values{"q": {"'; DROP TABLE users; --"}}) }, 403},
		{"nosql", `{"$gt":""} operator`, func() (*http.Response, error) { return post("/api/echo", `{"q":{"$gt":""}}`) }, 403},
		{"path", "../../etc/passwd", func() (*http.Response, error) { return get("/api/echo", url.Values{"file": {"../../etc/passwd"}}) }, 403},
		{"command", "; rm -rf /", func() (*http.Response, error) { return get("/api/echo", url.Values{"cmd": {"hi; rm -rf /"}}) }, 403},
		{"ssti", "Jinja2 {{7*7}}", func() (*http.Response, error) { return get("/api/echo", url.Values{"t": {"{{7*7}}"}}) }, 403},
		{"xxe", "DOCTYPE ENTITY", func() (*http.Response, error) {
			return post("/api/echo", `{"xml":"<!DOCTYPE foo [<!ENTITY xxe SYSTEM \"file:///etc/passwd\">]><foo>&xxe;</foo>"}`)
		}, 403},
	}

	fmt.Printf("\nArcis attack demo against %s\n%s\n", base(), repeat("-", 64))
	blocked, allowed, unexpected := 0, 0, 0
	for _, t := range tests {
		res, err := t.send()
		if err != nil {
			fmt.Printf("%sERR%s    %-8s %s: %v\n", colRed, colReset, t.category, t.label, err)
			unexpected++
			continue
		}
		_ = res.Body.Close()
		if res.StatusCode == t.expected {
			verb, note := "BLOCK", "Arcis denied"
			if t.expected == 200 {
				verb, note = "OK   ", "passed through"
			}
			fmt.Printf("%s%s%s  %-8s %s: %d (%s, as expected)\n", colGreen, verb, colReset, t.category, t.label, res.StatusCode, note)
			if t.expected == 200 {
				allowed++
			} else {
				blocked++
			}
		} else {
			label := "LEAK"
			if t.expected == 200 {
				label = "WHAT"
			}
			fmt.Printf("%s%s%s   %-8s %s: got %d, expected %d\n", colRed, label, colReset, t.category, t.label, res.StatusCode, t.expected)
			unexpected++
		}
	}
	fmt.Println(repeat("-", 64))
	plural := "s"
	if blocked == 1 {
		plural = ""
	}
	fmt.Printf("%s%d attack%s blocked%s, %d safe call%s passed, %s%d unexpected%s\n",
		colGreen, blocked, plural, colReset, allowed, pluralize(allowed), colYellow, unexpected, colReset)
	if unexpected > 0 {
		os.Exit(1)
	}
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
