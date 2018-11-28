FROM golang:alpine AS build

RUN mkdir -p /go/src/github.com/cloudlinker/kubecarve
COPY . /go/src/github.com/cloudlinker/kubecarve

WORKDIR /go/src/github.com/cloudlinker/kubecarve/podwatcher
RUN CGO_ENABLED=0 GOOS=linux go build 

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=build /go/src/github.com/cloudlinker/kubecarve/podwatcher/podwatcher /usr/local/bin/

ENTRYPOINT ["podwatcher"]
