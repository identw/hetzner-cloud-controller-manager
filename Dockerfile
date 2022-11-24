FROM docker.io/golang:1.18 as builder
LABEL org.opencontainers.image.source https://github.com/identw/hetzner-cloud-controller-manager
WORKDIR /maschine-controller/src
COPY ./go.mod .
COPY ./go.sum .
RUN go mod download
ADD . .
RUN CGO_ENABLED=0 go build -o hcloud-maschine-controller.bin  .

FROM docker.io/alpine:3.17.0 as certificates
RUN apk add --no-cache ca-certificates bash

FROM scratch
COPY --from=certificates /etc/ssl /etc/ssl
COPY --from=builder /maschine-controller/src/hcloud-maschine-controller.bin /hcloud-cloud-controller-manager
ENTRYPOINT ["/hcloud-cloud-controller-manager"]
