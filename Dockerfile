FROM golang:1.10-alpine as pre-builder

RUN set -ex \
    && apk add --no-cache --virtual .build-deps \
    gcc libc-dev git \
    && go get -u github.com/aws/aws-sdk-go/aws \
    && go get -u github.com/edimarlnx/go-plugins-helpers/volume
CMD ["/bin/false"]

FROM golang:1.10-alpine as builder
COPY --from=pre-builder /go/src /go/src
COPY . /go/src/github.com/edimarlnx/docker-ebs
WORKDIR /go/src/github.com/edimarlnx/docker-ebs
RUN go install --ldflags '-s -w -extldflags "-static"'

CMD ["/go/bin/docker-ebs"]

FROM alpine
RUN mkdir -p /run/docker/plugins /mnt/docker-ebs \
    && apk update \
    && apk add ca-certificates
COPY --from=builder /go/bin/docker-ebs ./docker-ebs-volume
CMD ["/docker-ebs-volume"]