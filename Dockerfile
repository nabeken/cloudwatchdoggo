# syntax=docker/dockerfile:1.3
FROM golang:1.17 as build-image

ENV GOMODCACHE /root/.cache/gomod

WORKDIR /go/src
COPY . ./

RUN --mount=type=cache,target=/root/.cache \
  go build -v -o /var/task/cloudwatchdoggo

FROM public.ecr.aws/lambda/go:1

COPY --from=build-image /go/bin/ /var/task/

# Command can be overwritten by providing a different command in the template directly.
CMD ["cloudwatchdoggo"]
