FROM docker.bottlenose.com/image/alpine-base

# Github personal access token to clone private repos
ARG pak

# ENV overwritten at runtime by env vars provided in Ansible role
ENV GITHUB_TOKEN=$pak

ADD ./ /nsqd-prometheus-exporter

WORKDIR /nsqd-prometheus-exporter

# Using go >= 1.6
RUN echo http://nl.alpinelinux.org/alpine/edge/community >> /etc/apk/repositories && \
    apk update && \
    apk add -U build-base file go git bash curl libstdc++ && \
    cd /nsqd-prometheus-exporter && \
    git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/" && \
    make && \
    apk del build-base go file git

EXPOSE 30000

CMD /nsqd-prometheus-exporter/nsqd-prometheus-exporter run
