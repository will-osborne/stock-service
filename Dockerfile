# We use alpine so we can also get the CA certs in this stage.
FROM golang:1.16.0-alpine3.13 as build

RUN apk --no-cache add ca-certificates

COPY src .
ENV GOPATH ""
ENV CGO_ENABLED=0
RUN go build -o /stocksvc


FROM scratch as service


COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /stocksvc /bin/stocksvc

ENTRYPOINT ["/bin/stocksvc"]