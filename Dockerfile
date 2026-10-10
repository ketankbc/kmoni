FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY server ./server
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/kmoni ./server
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/kmoni /kmoni
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/kmoni"]
