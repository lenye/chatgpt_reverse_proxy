FROM gcr.io/distroless/static-debian13:latest-amd64

COPY chatgpt_reverse_proxy /

ENTRYPOINT ["/chatgpt_reverse_proxy"]