package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// WebReverseProxyConfiguration is a coniguration for the ReverseProxy
type WebReverseProxyConfiguration struct {
	WhitelistedDomains []string
	ProxyHost          string
}

func main() {
	config := &WebReverseProxyConfiguration{
		WhitelistedDomains: []string{
			"rubygems.org",
			"www.rubygems.org",
			"packages-gitlab-com.s3-accelerate.amazonaws.com",
			"github.com",
		},
		ProxyHost: "127.0.0.1:8080",
	}

	proxy := NewWebReverseProxy(config)
	http.Handle("/", proxy)

	// Start the server
	http.ListenAndServe(":8080", nil)
}

func pullDomainAndPath(a string) (domain string, path string, err error) {
	data := strings.Split(a, "/")
	domain = data[1]
	path = "/" + strings.Join(data[2:], "/")

	return domain, path, err
}

func convertURLToProxy(config *WebReverseProxyConfiguration, u *url.URL) string {
	newURL := "http://" + config.ProxyHost + "/" + u.Host + u.Path
	if u.RawQuery != "" {
		newURL = newURL + "?" + u.RawQuery
	}

	return newURL
}

// NewWebReverseProxy returns a new ReverseProxy that routes
// traffic to it's intended target provided in the first url prefix. If the
// incoming request is http://$ProxyHost/rubygems.org/downloads/sawyer-0.8.1.gem then
// the target request will be for https://rubygems.org/downloads/sawyer-0.8.1.gem
func NewWebReverseProxy(config *WebReverseProxyConfiguration) *httputil.ReverseProxy {
	var err error

	director := func(req *http.Request) {
		req.URL.Scheme = "https"
		req.URL.Host, req.URL.Path, err = pullDomainAndPath(req.URL.Path)
		req.Host = req.URL.Host
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	responseDirector := func(res *http.Response) error {
		if location := res.Header.Get("Location"); location != "" {
			url, err := url.ParseRequestURI(location)
			if err != nil {
				fmt.Println("Error!")
				return err
			}

			newLocation := convertURLToProxy(config, url)
			res.Header.Set("Location", newLocation)
			res.Header.Set("X-Reverse-Proxy", "webreverseproxy")
		}
		return nil
	}

	return &httputil.ReverseProxy{
		Director:       director,
		ModifyResponse: responseDirector,
	}
}
