FROM alpine:3.18.3 AS certs
RUN apk --no-cache add ca-certificates && update-ca-certificates

FROM scratch
# for UI
EXPOSE 8080/tcp
# for output, should be RW
VOLUME /data
# for migrations, can be RO
VOLUME /migrations
# for form definitions, can be RO
VOLUME /configs
# pre-populate env
ENV FILES_PATH="/data" \
    CONFIGS="/configs" \
    DB_MIGRATIONS="/migrations" \
    DB_MIGRATE="true" \
    STORAGE="files" \
    HTTP_BIND="0.0.0.0:8080"

ENTRYPOINT ["/usr/bin/web-form"]
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ADD web-form /usr/bin/web-form