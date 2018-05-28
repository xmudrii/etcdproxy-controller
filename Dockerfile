FROM golang:1.10
COPY . /go/src/github.com/xmudrii/etcdproxy-controller
WORKDIR /go/src/github.com/xmudrii/etcdproxy-controller
RUN CGO_ENABLED=0 go build -o etcdproxy-controller main.go controller.go

FROM alpine:3.7
RUN apk add --no-cache ca-certificates
COPY --from=0 /go/src/github.com/xmudrii/etcdproxy-controller/etcdproxy-controller .
CMD ["./etcdproxy-controller"]