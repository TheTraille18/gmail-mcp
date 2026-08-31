FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /http ./cmd/http
	
FROM alpine:3.21
COPY --from=build /http /http
EXPOSE 3001
CMD ["/http"]
