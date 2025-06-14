FROM golang:1.24
LABEL maintainer="IST <team-infrastructure-services@scalingo.com>"

RUN go install github.com/cespare/reflex@latest

ADD . /go/src/github.com/Scalingo/acadock-monitoring
WORKDIR /go/src/github.com/Scalingo/acadock-monitoring
RUN go install -buildvcs=false github.com/Scalingo/acadock-monitoring/cmd/acadock-monitoring

CMD ["/go/bin/acadock-monitoring"]

EXPOSE 4244
