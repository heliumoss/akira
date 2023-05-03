FROM golang:bullseye

RUN apt-get update && apt-get install -y build-essential libvips-dev pkg-config

WORKDIR /akira

COPY go.mod go.sum ./

RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=1 GOOS=linux go build

EXPOSE 8000

CMD ["./akira"]