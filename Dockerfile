FROM debian:jessie

ADD esc-amd64 /esc

ENTRYPOINT ["/esc"]
