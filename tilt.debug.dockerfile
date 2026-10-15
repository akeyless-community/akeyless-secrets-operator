FROM golang:1.27.0@sha256:4013ae0f9e7994f8535c58c811f8f863fbed38b72e0d51e6592156f758d66146
WORKDIR /
COPY ./bin/external-secrets /external-secrets

RUN go install github.com/go-delve/delve/cmd/dlv@v1.22.0 && chmod +x /go/bin/dlv && mv /go/bin/dlv /

EXPOSE 30000

# dlv --listen=:30000 --api-version=2 --headless=true exec /app/build/api
ENTRYPOINT ["/dlv", "--listen=:30000", "--api-version=2", "--headless=true", "--continue=true", "--accept-multiclient=true", "exec", "/external-secrets", "--"]
