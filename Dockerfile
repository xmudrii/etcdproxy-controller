FROM golang:1.10
COPY . /go/src/github.com/xmudrii/etcdproxy-controller
WORKDIR /go/src/github.com/xmudrii/etcdproxy-controller
RUN make compile

FROM alpine:3.7
RUN apk add --no-cache ca-certificates
COPY --from=0 /go/src/github.com/xmudrii/etcdproxy-controller/bin/etcdproxy-controller .
CMD ["./etcdproxy-controller"]
