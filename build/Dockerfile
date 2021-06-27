FROM golang:1.17-rc-buster
RUN apt-get update
COPY main.go ./
ENTRYPOINT ["go", "run", "main.go"]