FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY gateway /usr/local/bin/gateway
RUN mkdir -p /data
ENV CLAWFIRM_DATA_DIR=/data
EXPOSE 9988
ENTRYPOINT ["gateway"]
CMD ["-addr", ":9988"]
