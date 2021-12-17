FROM golang:alpine as builder

RUN apk add -U --no-cache ca-certificates tzdata

WORKDIR /go/src/github.com/cooperspencer/gickup
COPY . .

RUN go get -d -v ./...
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o app .

FROM scratch as production
WORKDIR /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/cooperspencer/gickup/app /gickup/app
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
CMD ["./gickup/app"]
