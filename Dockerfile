FROM scratch

ADD esc-amd64 /esc

ENTRYPOINT ["/esc"]
