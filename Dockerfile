FROM golang:1.25
LABEL maintainer="IST <team-infrastructure-services@scalingo.com>"

RUN go install github.com/cespare/reflex@latest

ADD . /go/src/github.com/Scalingo/acadock-monitoring
WORKDIR /go/src/github.com/Scalingo/acadock-monitoring

RUN go build -buildvcs=false -o /go/bin/acadock-monitoring ./cmd/acadock-monitoring

CMD ["/go/bin/acadock-monitoring"]

EXPOSE 4244
