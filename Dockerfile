FROM golang:1.23.4 AS builder

WORKDIR /app

COPY . .

RUN go mod download

COPY .env .

RUN CGO_ENABLED=0 GOOS=linux go build -C ./cmd -a -installsuffix cgo -o ./../myapp .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/myapp .
COPY --from=builder /app/casbin/model.conf ./casbin/
COPY --from=builder /app/casbin/policy.csv ./casbin/
COPY --from=builder /app/doc/swagger/index.html ./doc/swagger/
COPY --from=builder /app/doc/swagger/swagger_docs.swagger.json ./doc/swagger/
COPY --from=builder /app/app.log ./
COPY --from=builder /app/.env .

EXPOSE 8080

CMD ["./myapp"]