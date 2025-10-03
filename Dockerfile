# syntax=docker/dockerfile:1.19
FROM golang:1.25 AS build-image

WORKDIR /go/src
COPY . ./

RUN go env
RUN go build -tags lambda.norpc -v -o ../bin/cloudwatchdoggo

FROM public.ecr.aws/lambda/provided:al2023

COPY --from=build-image /go/bin/cloudwatchdoggo ./

# Command can be overwritten by providing a different command in the template directly.
CMD ["cloudwatchdoggo"]
