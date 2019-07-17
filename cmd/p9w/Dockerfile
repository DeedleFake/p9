FROM golang AS build

RUN mkdir -p /build
RUN GOPATH=/build CGO_ENABLED=0 go get -u -v github.com/DeedleFake/p9/cmd/p9w

FROM scratch

COPY --from=build /build/bin/p9w /

EXPOSE 8080

ENTRYPOINT ["/p9w"]
