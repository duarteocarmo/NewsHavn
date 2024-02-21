# builder
FROM golang:1.22 as builder

COPY . /src/myapp
WORKDIR /src/myapp

RUN --mount=type=cache,target=/root/.cache/go-build \
	--mount=type=cache,target=/go/pkg \
	go build -ldflags '-s -w -extldflags "-static"' -tags osusergo,netgo,sqlite_omit_load_extension -o /usr/local/bin/myapp /src/myapp/cmd/server/main.go

ADD https://github.com/benbjohnson/litestream/releases/download/v0.3.8/litestream-v0.3.8-linux-amd64-static.tar.gz /tmp/litestream.tar.gz
RUN tar -C /usr/local/bin -xzf /tmp/litestream.tar.gz

# runner
FROM alpine

COPY --from=builder /usr/local/bin/myapp /usr/local/bin/myapp
COPY --from=builder /usr/local/bin/litestream /usr/local/bin/litestream

RUN apk add bash

EXPOSE 8080

COPY config/litestream.yml /etc/litestream.yml
COPY scripts/run.sh /scripts/run.sh

CMD [ "/scripts/run.sh" ]
