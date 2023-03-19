package tools

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

var originalUrlResolver = make(map[string]*url.URL)
var mutex = &sync.Mutex{}

// ProxyRequestHandler intercepts requests to CodeArtifact and add the Authorization header + correct Host header
func ProxyRequestHandler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		mutex.Lock()
		// Store the original host header for each request
		originalUrlResolver[r.RemoteAddr] = r.URL
		originalUrlResolver[r.RemoteAddr].Host = r.Host
		originalUrlResolver[r.RemoteAddr].Scheme = r.URL.Scheme

		if r.Header.Get("X-Forwarded-Proto") == "https" {
			originalUrlResolver[r.RemoteAddr].Scheme = "https"
		} else {
			originalUrlResolver[r.RemoteAddr].Scheme = "http"
		}
		mutex.Unlock()

		// Override the Host header with the CodeArtifact Host
		u, _ := url.Parse(CodeArtifactAuthInfo.Url)
		r.Host = u.Host

		// Set the Authorization header with the CodeArtifact Authorization Token
		r.SetBasicAuth("aws", CodeArtifactAuthInfo.AuthorizationToken)

		log.Printf("REQ: %s %s \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.URL.RequestURI(), r.UserAgent())

		log.Printf("Sending request to %s%s", strings.Trim(CodeArtifactAuthInfo.Url, "/"), r.URL.RequestURI())

		p.ServeHTTP(w, r)
	}
}

func ProxyResponseHandler() func(*http.Response) error {
	return func(r *http.Response) error {
		log.Printf("Received response from %s", r.Request.URL.String())
		log.Printf("RES: %s \"%s\" %d \"%s\" \"%s\"", r.Request.RemoteAddr, r.Request.Method, r.StatusCode, r.Request.RequestURI, r.Request.UserAgent())

		contentType := r.Header.Get("Content-Type")

		mutex.Lock()
		originalUrl := originalUrlResolver[r.Request.RemoteAddr]
		delete(originalUrlResolver, r.Request.RemoteAddr)
		mutex.Unlock()

		u, _ := url.Parse(CodeArtifactAuthInfo.Url)
		hostname := u.Host + ":443"

		// Rewrite the 301 to point from CodeArtifact URL to the proxy instead..
		if r.StatusCode == 301 || r.StatusCode == 302 {
			location, _ := r.Location()

			location.Host = originalUrl.Host
			location.Scheme = originalUrl.Scheme
			location.Path = strings.Replace(location.Path, u.Path, "", 1)

			r.Header.Set("Location", location.String())
		}

		// Do some quick fixes to the HTTP response for NPM install requests
		if strings.HasPrefix(r.Request.UserAgent(), "npm") {

			// Respond to only requests that respond with JSON
			// There might eventually be additional headers i don't know about?
			if !strings.Contains(contentType, "application/json") && !strings.Contains(contentType, "application/vnd.npm.install-v1+json") {
				return nil
			}

			var body io.ReadCloser

			if r.Header.Get("Content-Encoding") == "gzip" {
				body, _ = gzip.NewReader(r.Body)
				r.Header.Del("Content-Encoding")
			} else {
				body = r.Body
			}

			// replace any instances of the CodeArtifact URL with the local URL
			oldContentResponse, _ := ioutil.ReadAll(body)
			oldContentResponseStr := string(oldContentResponse)

			resolvedHostname := strings.Replace(CodeArtifactAuthInfo.Url, u.Host, hostname, -1)
			newUrl := fmt.Sprintf("%s://%s/", originalUrl.Scheme, originalUrl.Host)

			newResponseContent := strings.Replace(oldContentResponseStr, resolvedHostname, newUrl, -1)
			newResponseContent = strings.Replace(newResponseContent, CodeArtifactAuthInfo.Url, newUrl, -1)

			r.Body = ioutil.NopCloser(strings.NewReader(newResponseContent))
			r.ContentLength = int64(len(newResponseContent))
			r.Header.Set("Content-Length", strconv.Itoa(len(newResponseContent)))
		}

		return nil
	}

}

// ProxyInit initialises the CodeArtifact proxy and starts the HTTP listener
func ProxyInit() {
	remote, err := url.Parse(CodeArtifactAuthInfo.Url)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)

	proxy.ModifyResponse = ProxyResponseHandler()

	http.HandleFunc("/", ProxyRequestHandler(proxy))
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
