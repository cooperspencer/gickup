FROM golang:alpine as builder

# Install dependencies for copy
RUN apk add -U --no-cache ca-certificates tzdata git

# Use an valid GOPATH and copy the files
WORKDIR /go/src/github.com/cooperspencer/gickup
COPY . .

# Fetching dependencies and build the app
RUN go get -d -v ./...
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o app .

# Use scratch as production environment -> Small builds
FROM scratch as production
WORKDIR /
# Copy valid SSL certs from the builder for fetching github/gitlab/...
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copy zoneinfo for getting the right cron timezone
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
# Copy the main executable from the builder
COPY --from=builder /go/src/github.com/cooperspencer/gickup/app /gickup/app

ENTRYPOINT [ "/gickup/app" ]
CMD [ "/gickup/conf.yml" ]
