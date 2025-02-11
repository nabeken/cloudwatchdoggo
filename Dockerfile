# syntax=docker/dockerfile:1.13
FROM golang:1.23 AS build-image

ENV GOMODCACHE=/root/.cache/gomod

WORKDIR /go/src
COPY . ./

RUN go env

RUN --mount=type=cache,target=/root/.cache \
  go build -tags lambda.norpc -v -o ../bin/cloudwatchdoggo

FROM public.ecr.aws/lambda/provided:al2023

COPY --from=build-image /go/bin/cloudwatchdoggo ./

# Command can be overwritten by providing a different command in the template directly.
CMD ["cloudwatchdoggo"]
