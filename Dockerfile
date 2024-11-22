FROM golang:1.23-alpine

RUN apk add --no-cache git

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN go build -ldflags "-s -w" -o /usr/local/bin/app

CMD ["app"]