# Dockerfile — playground-viewer (read-only DriftReport live view).
# Built from the repo root so it shares the project's go.mod / go.sum:
#   docker build -f examples/playground/viewer/Dockerfile -t cha-playground-viewer:dev .
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
    -o /out/playground-viewer ./examples/playground/viewer

FROM gcr.io/distroless/static-debian12:nonroot
USER nonroot:nonroot
COPY --from=build /out/playground-viewer /playground-viewer
EXPOSE 8080
ENTRYPOINT ["/playground-viewer"]
