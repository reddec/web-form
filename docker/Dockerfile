FROM alpine:3.18.3 AS build
RUN apk --no-cache add ca-certificates tzdata && update-ca-certificates

FROM scratch
# for UI
EXPOSE 8080/tcp
# for output, should be RW
VOLUME /data
# for migrations, can be RO
VOLUME /migrations
# for form definitions, can be RO
VOLUME /configs
# for user-defined static files avaiable via /assets/
VOLUME /assets
# pre-populate env
ENV FILES_PATH="/data" \
    CONFIGS="/configs" \
    DB_MIGRATIONS="/migrations" \
    DB_MIGRATE="true" \
    STORAGE="files" \
    HTTP_BIND="0.0.0.0:8080" \
    HTTP_ASSETS="/assets"

ENTRYPOINT ["/usr/bin/web-form"]
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ADD web-form /usr/bin/web-form