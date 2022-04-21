FROM golang:1.17 AS builder
RUN apt update && apt install -y git
COPY . /usr/src
WORKDIR /usr/src
RUN go generate
RUN CGO_ENABLED=0 go build -o s3logd main.go

FROM gcr.io/distroless/static
COPY --from=builder /usr/src/s3logd /s3logd
CMD [ "/s3logd", "stream" ]