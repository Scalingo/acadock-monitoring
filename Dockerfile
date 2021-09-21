FROM golang:1.16
LABEL maintainer="leo@scalingo.com"

RUN go get github.com/cespare/reflex

ADD . /go/src/github.com/Scalingo/acadock-monitoring
WORKDIR /go/src/github.com/Scalingo/acadock-monitoring
RUN go install github.com/Scalingo/acadock-monitoring/cmd/acadock-monitoring

CMD ["/go/bin/acadock-monitoring"]

EXPOSE 4244
