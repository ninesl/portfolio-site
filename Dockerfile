FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN go tool templ generate

# Go's linker only exposes these size flags as shorthand: -s strips the symbol table, -w omits DWARF debug info.
RUN CGO_ENABLED=0 go build \
    --trimpath \
    --buildvcs=false \
    --mod=readonly \
    --ldflags="-s -w" \
    -o /usr/local/bin/ninescoding \
    .

FROM alpine:latest AS launcher

RUN apk add --no-cache ca-certificates

COPY --from=builder /usr/local/bin/ninescoding /usr/local/bin/ninescoding
:L:w

WORKDIR /
EXPOSE 8080
CMD ["ninescoding"]
