package tools

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
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
