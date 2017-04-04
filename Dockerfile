FROM alpine:3.5

ADD ./ /go/src/github.com/adamweiner/nsqd-prometheus-exporter

RUN apk update && \
    apk add -U build-base go git curl libstdc++ && \
    cd /go/src/github.com/adamweiner/nsqd-prometheus-exporter && \
    GOPATH=/go make && \
    apk del build-base go git

EXPOSE 30000

CMD /go/src/github.com/adamweiner/nsqd-prometheus-exporter
