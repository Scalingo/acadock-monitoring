FROM golang:latest

MAINTAINER leo@scalingo.com

RUN go get github.com/tools/godep

ADD . /go/src/github.com/Scalingo/acadock-monitoring
RUN cd /go/src/github.com/Scalingo/acadock-monitoring/server && \
    godep go install && \
    cd /go/src/github.com/Scalingo/acadock-monitoring/runner/acadock-monitoring-ns-netstat && \
    godep go install

ENV RUNNER_DIR=/go/bin

CMD /go/bin/server

EXPOSE 4244
