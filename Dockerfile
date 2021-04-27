FROM golang:1.15-alpine AS build

WORKDIR /go/src/standup-store-monitoring
COPY . .

RUN mkdir -p /go/bin/standup-store-monitoring
RUN go build -mod=vendor -o /go/bin/standup-store-monitoring/app cmd/standupstoremonitoring/main.go

FROM alpine:latest AS bin

ENV CHECK_INTERVAL=${CHECK_INTERVAL}
ENV CHAT_ID=${CHAT_ID}
ENV TOKEN=${TOKEN}

COPY --from=build /go/bin/standup-store-monitoring/app .
CMD ["./app"]