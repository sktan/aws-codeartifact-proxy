package tools

import (
	"compress/gzip"
	"fmt"
	"io"

	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
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

		// Override the Host header with the CodeArtifact Host
		u, _ := url.Parse(CodeArtifactAuthInfo.Url)
		r.Host = u.Host

		// Set the Authorization header with the CodeArtifact Authorization Token
		r.SetBasicAuth("aws", CodeArtifactAuthInfo.AuthorizationToken)

		log.Printf("REQ: %s %s \"%s\" \"%s\"", r.RemoteAddr, r.Method, r.URL.RequestURI(), r.UserAgent())

		log.Printf("Sending request to %s%s", strings.Trim(CodeArtifactAuthInfo.Url, "/"), r.URL.RequestURI())
		mutex.Unlock()

		p.ServeHTTP(w, r)
	}
}

// Handles the response back to the client once intercepted from CodeArtifact
func ProxyResponseHandler() func(*http.Response) error {
	return func(r *http.Response) error {
		log.Printf("Received %d response from %s", r.StatusCode, r.Request.URL.String())
		log.Printf("RES: %s \"%s\" %d \"%s\" \"%s\"", r.Request.RemoteAddr, r.Request.Method, r.StatusCode, r.Request.RequestURI, r.Request.UserAgent())

		contentType := r.Header.Get("Content-Type")

		mutex.Lock()
		originalUrl := originalUrlResolver[r.Request.RemoteAddr]
		delete(originalUrlResolver, r.Request.RemoteAddr)

		u, _ := url.Parse(CodeArtifactAuthInfo.Url)
		hostname := u.Host + ":443"
		mutex.Unlock()

		// Rewrite the 301 to point from CodeArtifact URL to the proxy instead..
		if r.StatusCode == 301 || r.StatusCode == 302 {
			location, _ := r.Location()

			// Only attempt to rewrite the location if the host matches the CodeArtifact host
			// Otherwise leave the original location intact (e.g a redirect to a S3 presigned URL)
			if location.Host == u.Host {
				location.Host = originalUrl.Host
				location.Scheme = originalUrl.Scheme
				location.Path = strings.Replace(location.Path, u.Path, "", 1)

				r.Header.Set("Location", location.String())
			}
		}

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
		oldContentResponse, _ := io.ReadAll(body)
		oldContentResponseStr := string(oldContentResponse)

		mutex.Lock()
		resolvedHostname := strings.ReplaceAll(CodeArtifactAuthInfo.Url, u.Host, hostname)
		newUrl := fmt.Sprintf("%s://%s/", originalUrl.Scheme, originalUrl.Host)

		newResponseContent := strings.ReplaceAll(oldContentResponseStr, resolvedHostname, newUrl)
		newResponseContent = strings.ReplaceAll(newResponseContent, CodeArtifactAuthInfo.Url, newUrl)
		mutex.Unlock()

		r.Body = io.NopCloser(strings.NewReader(newResponseContent))
		r.ContentLength = int64(len(newResponseContent))
		r.Header.Set("Content-Length", strconv.Itoa(len(newResponseContent)))

		return nil
	}

}

// HealthCheckHandler responds to health probes without proxying to
// CodeArtifact. It returns 200 OK when a valid authorization token is held and
// 503 Service Unavailable otherwise, so orchestrators can tell when the proxy
// is unable to serve requests.
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if CodeArtifactAuthInfo.IsTokenValid() {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintln(w, "CodeArtifact authorization token is not valid")
	}
}

// ProxyInit initialises the CodeArtifact proxy and starts the HTTP listener
func ProxyInit() {
	remote, err := url.Parse(CodeArtifactAuthInfo.Url)
	if err != nil {
		panic(err)
	}

	// Get port from LISTEN_PORT environment variable. If not set, default to 8080.
	port := getEnv("LISTEN_PORT", "8080")

	proxy := httputil.NewSingleHostReverseProxy(remote)

	proxy.ModifyResponse = ProxyResponseHandler()

	proxyHandler := ProxyRequestHandler(proxy)

	// The health check endpoint is opt-in: it is only enabled when
	// HEALTHCHECK_PATH is set (e.g. /actuator/health). When enabled, a GET on
	// that path is served as a health check rather than being proxied. All
	// other requests (including non-GET requests to the health path) are
	// proxied as usual.
	if healthPath, ok := os.LookupEnv("HEALTHCHECK_PATH"); ok && healthPath != "" {
		log.Printf("Health check enabled on GET %s", healthPath)
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == healthPath && r.Method == http.MethodGet {
				HealthCheckHandler(w, r)
				return
			}
			proxyHandler(w, r)
		})
	} else {
		http.HandleFunc("/", proxyHandler)
	}
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
