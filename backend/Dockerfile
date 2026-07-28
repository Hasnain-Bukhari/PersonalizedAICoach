FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/coach-api ./cmd/api

FROM alpine:3.21

RUN addgroup -S coach && adduser -S -G coach coach
COPY --from=build /out/coach-api /usr/local/bin/coach-api
USER coach
EXPOSE 8080
ENTRYPOINT ["coach-api"]
