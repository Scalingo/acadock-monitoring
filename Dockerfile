FROM golang:1.17
LABEL maintainer="Léo Unbekandt <leo@scalingo.com>"

RUN go install github.com/cespare/reflex@latest

ADD . /go/src/github.com/Scalingo/acadock-monitoring
WORKDIR /go/src/github.com/Scalingo/acadock-monitoring
RUN go install github.com/Scalingo/acadock-monitoring/cmd/acadock-monitoring

CMD ["/go/bin/acadock-monitoring"]

EXPOSE 4244
