FROM gcr.io/distroless/static-debian12
COPY chatgpt_reverse_proxy /
ENTRYPOINT ["/chatgpt_reverse_proxy"]