FROM alpine

# ADD Curl
#when you need it

# create and use non-root user
RUN adduser \
  --disabled-password \
  --home /app \
  --gecos '' app \
  && chown -R app /app

USER app

WORKDIR /app
COPY ./.output/app ./

ENTRYPOINT [ "./ascraper" ]
