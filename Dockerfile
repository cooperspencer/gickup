FROM golang:1.25-alpine AS builder

# Install dependencies for copy
RUN apk add -U --no-cache ca-certificates tzdata git

# Use an valid GOPATH and copy the files
WORKDIR /go/src/github.com/cooperspencer/gickup
COPY go.mod .
COPY go.sum .
RUN go mod tidy
COPY . .

# Fetching dependencies and build the app
RUN go get -d -v ./...
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o gickup .

# Use alpine as production environment -> Small builds
FROM alpine:3.20 AS production
RUN apk add -U --no-cache ca-certificates tzdata git git-lfs openssh
RUN git lfs install

WORKDIR /
# Copy the main executable from the builder
COPY --from=builder /go/src/github.com/cooperspencer/gickup/gickup /gickup/gickup

ENTRYPOINT [ "/gickup/gickup" ]
CMD [ "/gickup/conf.yml" ]
