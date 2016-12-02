FROM golang:1.7.1

MAINTAINER leo@scalingo.com

ADD . /go/src/github.com/Scalingo/acadock-monitoring
RUN cd /go/src/github.com/Scalingo/acadock-monitoring/server && \
    go install && \
    cd /go/src/github.com/Scalingo/acadock-monitoring/runner/acadock-monitoring-ns-netstat && \
    go install

ENV RUNNER_DIR=/go/bin

CMD ["/go/bin/server"]

EXPOSE 4244
