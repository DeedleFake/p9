FROM golang AS build

RUN mkdir -p /build
RUN GOPATH=/build CGO_ENABLED=0 go get -u -v github.com/DeedleFake/p9/cmd/p9

FROM scratch

COPY --from=build /build/bin/p9 /

EXPOSE 564

ENTRYPOINT ["/p9"]
