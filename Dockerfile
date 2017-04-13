FROM golang:1.7.1

MAINTAINER leo@scalingo.com

ADD . /go/src/github.com/Scalingo/acadock-monitoring
RUN go install github.com/Scalingo/acadock-monitoring/cmd/acadock-monitoring

CMD ["/go/bin/acadock-monitoring"]

EXPOSE 4244
