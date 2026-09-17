FROM docker.io/library/alpine:3.22
RUN apk add --no-cache 7zip
COPY bench-image /usr/local/bin/bench-image
