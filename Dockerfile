FROM golang:1.18-alpine as app-builder
WORKDIR /go/src/app

COPY src .

RUN CGO_ENABLED=0 go install -ldflags '-extldflags "-static"' -tags timetzdata

# Build actual image with the compiled app
FROM scratch

LABEL maintainer="git@sktan.com"

COPY --from=app-builder /go/bin/aws-codeartifact-proxy /aws-codeartifact-proxy
COPY --from=app-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/aws-codeartifact-proxy"]
