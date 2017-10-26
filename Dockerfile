FROM golang:1.9.1-alpine3.6

ADD ./ /go/src/github.com/adamweiner/nsqd-prometheus-exporter

RUN apk update && \
    apk add -U build-base git && \
    cd /go/src/github.com/adamweiner/nsqd-prometheus-exporter && \
    GOPATH=/go make && \
    apk del build-base git

EXPOSE 30000

ENTRYPOINT ["/go/src/github.com/adamweiner/nsqd-prometheus-exporter/nsqd-prometheus-exporter"]
