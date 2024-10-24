# syntax=docker/dockerfile:1.4
FROM --platform=${BUILDPLATFORM} golang:1.23-alpine3.20 as builder
ENV GOPATH="/go"
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace
RUN apk add --no-cache bash
SHELL [ "/bin/bash", "-euo", "pipefail", "-c" ]
COPY go.mod go.mod
COPY go.sum go.sum
COPY go.wor[k] go.work
RUN --mount=type=cache,id=go-cache,target=/go/pkg <<eot
    go mod download
eot

COPY . .

RUN --mount=type=cache,id=oobit-go-cache,target=/go/pkg <<eot
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o gokeenapi
eot

FROM alpine:3.20 as final
WORKDIR /gokeenapi
ENV PATH="${PATH}:/gokeenapi"
COPY --from=builder /workspace/gokeenapi ./gokeenapi
ENTRYPOINT [ "gokeenapi" ]
CMD [ "--help" ]