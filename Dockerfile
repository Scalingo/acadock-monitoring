FROM golang:1.7.1

MAINTAINER leo@scalingo.com

ADD . /go/src/github.com/Scalingo/acadock-monitoring
RUN cd /go/src/github.com/Scalingo/acadock-monitoring && \
    go install cmd/...

CMD ["/go/bin/server"]

EXPOSE 4244
