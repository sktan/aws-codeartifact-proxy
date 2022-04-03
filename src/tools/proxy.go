package tools

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

// ProxyRequestHandler intercepts requests to CodeArtifact and add the Authorization header + correct Host header
func ProxyRequestHandler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("RES: %s \"%s\" %d \"%s\" \"%s\"", r.Request.RemoteAddr, r.Request.Method, r.StatusCode, r.Request.RequestURI, r.Request.UserAgent())

		contentType := r.Header.Get("Content-Type")
		log.Printf("%s", contentType)

		// Do some quick fixes to the HTTP response for NPM install requests
		// TODO: Get this actually working, it looks like the JSON responses provide the correct URLs via CURL, but not when using npm against it.
		if strings.HasPrefix(r.Request.UserAgent(), "npm") {
			if !strings.Contains(contentType, "application/json") {
				return nil
			}

			// replace any instances of the CodeArtifact URL with the local URL
			oldContentResponse, _ := ioutil.ReadAll(r.Body)
			oldContentResponseStr := string(oldContentResponse)

			u, _ := url.Parse(CodeArtifactAuthInfo.Url)
			hostname := u.Host + ":443"

			resolvedHostname := strings.Replace(CodeArtifactAuthInfo.Url, u.Host, hostname, -1)
			newUrl := fmt.Sprintf("http://%s/", "localhost")
			log.Printf("URI conversion %s -> %s", resolvedHostname, newUrl)

			newResponseContent := strings.Replace(oldContentResponseStr, resolvedHostname, newUrl, -1)
			newResponseContent = strings.Replace(newResponseContent, CodeArtifactAuthInfo.Url, newUrl, -1)

			log.Print(newResponseContent)

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
