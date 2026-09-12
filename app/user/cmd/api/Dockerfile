FROM golang:1.26.1

WORKDIR /app

COPY . .

RUN go env -w GOPROXY=https://goproxy.cn,direct

RUN go install github.com/go-delve/delve/cmd/dlv@latest

EXPOSE 18888

CMD ["dlv", "debug", "./app/user/cmd/api", "--headless", "--listen=:18888", "--api-version=2", "--accept-multiclient", \
    "--", "-buildvcs=false", "-f", "./deploy/config/user-api.yml"]
