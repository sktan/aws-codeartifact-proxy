FROM golang:1.25-alpine as app-builder
WORKDIR /go/src/app

RUN echo "nobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin" > /etc/passwd.nobody

COPY src .

RUN CGO_ENABLED=0 go install -ldflags '-extldflags "-static"' -tags timetzdata

# Build actual image with the compiled app
FROM scratch

LABEL maintainer="git@sktan.com"

COPY --from=app-builder /etc/passwd.nobody /etc/passwd
COPY --from=app-builder /go/bin/aws-codeartifact-proxy /aws-codeartifact-proxy
COPY --from=app-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER 65534

ENTRYPOINT ["/aws-codeartifact-proxy"]
