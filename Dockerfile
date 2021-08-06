FROM golang:1.16-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN apk add --no-cache ffmpeg

RUN go build -o /server

EXPOSE 8080

CMD [ "/server" ]