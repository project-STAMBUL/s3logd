FROM golang:1.17 AS builder
RUN apt update && apt install -y git
COPY . /usr/src
WORKDIR /usr/src
RUN go generate
RUN CGO_ENABLED=0 go build -o s3logd main.go

FROM busybox:1.35.0-uclibc as busybox
FROM gcr.io/distroless/static
# Now copy the static shell into base image.
COPY --from=busybox /bin/sh /bin/sh
# You may also copy all necessary executables into distroless image.
COPY --from=busybox /bin/mkdir /bin/mkdir
COPY --from=busybox /bin/cat /bin/cat
COPY --from=builder /usr/src/s3logd /s3logd
CMD [ "/s3logd", "stream" ]
