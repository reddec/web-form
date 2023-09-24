FROM busybox
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
ADD web-form /usr/bin/web-form