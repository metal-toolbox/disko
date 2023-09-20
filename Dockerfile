FROM alpine:latest

ENTRYPOINT ["/usr/sbin/disko"]

COPY disko /usr/sbin/disko
RUN chmod +x /usr/sbin/disko
