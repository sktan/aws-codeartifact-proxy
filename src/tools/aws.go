package tools

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codeartifact"
	"github.com/aws/aws-sdk-go-v2/service/codeartifact/types"
)

type CodeArtifactAuthInfoStruct struct {
	Url                string
	AuthorizationToken string
	LastAuth           time.Time
}

var CodeArtifactAuthInfo = &CodeArtifactAuthInfoStruct{}

// Authenticate performs the authentication against CodeArtifact and caches the credentials
func Authenticate() {
	log.Printf("Authenticating against CodeArtifact")

	// Authenticate against CodeArtifact
	cfg, cfgErr := config.LoadDefaultConfig(context.TODO())
	if cfgErr != nil {
		log.Fatalf("unable to load SDK config, %v", cfgErr)
	}
	svc := codeartifact.NewFromConfig(cfg)

	codeArtDomain := aws.String(os.Getenv("CODEARTIFACT_DOMAIN"))
	codeArtOwner, codeArtOwnerFound := os.LookupEnv("CODEARTIFACT_OWNER")
	codeArtRepos := aws.String(os.Getenv("CODEARTIFACT_REPO"))

	// Resolve Package Format from the environment variable (defaults to pypi)
	codeArtTypeS, found := os.LookupEnv("CODEARTIFACT_TYPE")
	if !found || codeArtTypeS == "" {
		codeArtTypeS = "pypi"
	}
	var codeArtTypeT types.PackageFormat
	if codeArtTypeS == "pypi" {
		codeArtTypeT = types.PackageFormatPypi
	} else if codeArtTypeS == "maven" {
		codeArtTypeT = types.PackageFormatMaven
	} else if codeArtTypeS == "npm" {
		codeArtTypeT = types.PackageFormatNpm
	} else if codeArtTypeS == "nuget" {
		codeArtTypeT = types.PackageFormatNuget
	}

	// Create the input for the CodeArtifact API
	authInput := &codeartifact.GetAuthorizationTokenInput{
		DurationSeconds: aws.Int64(3600),
		Domain:          codeArtDomain,
	}
	if codeArtOwnerFound {
		authInput.DomainOwner = aws.String(codeArtOwner)
	}
	authResp, authErr := svc.GetAuthorizationToken(context.TODO(), authInput)
	if authErr != nil {
		log.Fatalf("unable to get authorization token, %v", authErr)
	}
	log.Printf("Authorization successful")

	mutex.Lock()
	CodeArtifactAuthInfo.AuthorizationToken = *authResp.AuthorizationToken
	CodeArtifactAuthInfo.LastAuth = time.Now()

	// Get the URL for the CodeArtifact Service
	urlInput := &codeartifact.GetRepositoryEndpointInput{
		Domain:     codeArtDomain,
		Format:     codeArtTypeT,
		Repository: codeArtRepos,
	}
	if codeArtOwnerFound {
		urlInput.DomainOwner = aws.String(codeArtOwner)
	}

	urlResp, urlErr := svc.GetRepositoryEndpoint(context.TODO(), urlInput)
	if urlErr != nil {
		log.Fatalf("unable to get repository endpoint, %v", urlErr)
	}
	CodeArtifactAuthInfo.Url = *urlResp.RepositoryEndpoint
	mutex.Unlock()

	log.Printf("Requests will now be proxied to %s", CodeArtifactAuthInfo.Url)
}

// CheckReauth checks if we have not yet authenticated, or need to authenticate within the next 15 minutes
func CheckReauth() {
	for {
		timeSince := time.Since(CodeArtifactAuthInfo.LastAuth).Minutes()
		// Panic and shut down the proxy if we couldn't reauthenticate within the 15 minute window for some reason.
		if timeSince > float64(60) {
			log.Panic("Was unable to re-authenticate prior to our token expiring, shutting down proxty...")
		}

		if CodeArtifactAuthInfo.AuthorizationToken == "" || timeSince > float64(45) {
			log.Printf("%f minutes until the CodeArtifact token expires, attempting a reauth.", 60-timeSince)
			Authenticate()
		}
		// Sleep for 15 seconds for the next check
		time.Sleep(15 * time.Second)
	}
}
