FROM squidfunk/mkdocs-material:9.6.16@sha256:bb7b015690d9fb5ef0dbc98ca3520f153aa43129fb96aec5ca54c9154dc3b729

COPY action.sh /action.sh

RUN apk add --no-cache bash \
  && chmod +x /action.sh

ENTRYPOINT ["/action.sh"]
