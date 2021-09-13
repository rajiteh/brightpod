FROM golang:1.16
WORKDIR /src/brightpod/
COPY ./ ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o brightpod .

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /src/brightpod/brightpod ./
CMD ["./brightpod"]  