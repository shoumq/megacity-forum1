FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -o /out/megacity-api .

FROM alpine:3.21
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build /out/megacity-api ./megacity-api
RUN mkdir -p /app/data && chown -R app:app /app
USER app
EXPOSE 8080
CMD ["./megacity-api"]
