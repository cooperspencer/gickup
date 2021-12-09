FROM golang:alpine as builder

WORKDIR /go/src/github.com/cooperspencer/gickup
COPY . .

RUN go get -d -v ./...
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o app .

FROM scratch as production
WORKDIR /
COPY --from=builder /go/src/github.com/cooperspencer/gickup/app /gickup/app
CMD ["./gickup/app"]
