FROM golang:1.26
LABEL maintainer="IST <team-infrastructure-services@scalingo.com>"

RUN apt-get update \
  && apt-get install -y --no-install-recommends iproute2 \
  && rm -rf /var/lib/apt/lists/*

RUN go install github.com/cespare/reflex@latest

ADD . /go/src/github.com/Scalingo/acadock-monitoring
WORKDIR /go/src/github.com/Scalingo/acadock-monitoring

RUN go build -buildvcs=false -o /go/bin/acadock-monitoring ./cmd/acadock-monitoring

CMD ["/go/bin/acadock-monitoring"]

EXPOSE 4244
