# syntax=docker/dockerfile:1.4
FROM --platform=${BUILDPLATFORM} golang:1.25-alpine3.21 as builder
ENV GOPATH="/go"
ARG TARGETOS
ARG TARGETARCH
ARG GOKEENAPI_VERSION="undefined"
ARG GOKEENAPI_BUILDDATE="undefined"
WORKDIR /workspace
RUN apk add --no-cache bash
SHELL [ "/bin/bash", "-euo", "pipefail", "-c" ]
COPY go.mod go.mod
COPY go.sum go.sum
COPY go.wor[k] go.work
RUN --mount=type=cache,id=golang,target=/go/pkg/mod/ <<eot
    go mod download
eot

COPY . .

RUN --mount=type=cache,id=golang,target=/go/pkg/mod/ <<eot
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-X \"github.com/noksa/gokeenapi/internal/gokeenversion.version=${GOKEENAPI_VERSION}\" -X \"github.com/noksa/gokeenapi/internal/gokeenversion.buildDate=${GOKEENAPI_BUILDDATE}\"" -o gokeenapi
eot

FROM alpine:3.21 as final
WORKDIR /opt/gokeenapi
COPY --from=builder /workspace/gokeenapi ./gokeenapi
ENV PATH="${PATH}:/opt/gokeenapi"
ENV GOKEENAPI_INSIDE_DOCKER=true
RUN mkdir -p /etc/gokeenapi && echo "{}" > /etc/gokeenapi/.gokeenapi
VOLUME [ "/etc/gokeenapi" ]
ENTRYPOINT [ "gokeenapi" ]
CMD [ "--help" ]


# docker buildx build --platform "linux/arm64,linux/amd64" -t "${TAG}" --pull --push --build-arg=GOKEENAPI_VERSION="${VERSION}" --build-arg=GOKEENAPI_BUILDDATE="$(date)" .