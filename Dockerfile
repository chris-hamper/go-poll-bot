################
## Build stage
################
FROM golang:1.11-alpine3.8 AS build
RUN apk --no-cache add git gcc musl-dev

WORKDIR /src
ADD . .
RUN go build -o ../bin/go-app


################
## Final stage
################
FROM alpine:3.8

WORKDIR /app
COPY --from=build /bin/go-app .

EXPOSE 3000
ENTRYPOINT ./go-app
