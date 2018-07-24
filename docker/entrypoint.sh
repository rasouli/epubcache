#! /bin/sh
echo ">> Waiting for database to accept connection..."
WAIT=0
while ! nc -z $EPUB_CACHE_CONFIG_DB_HOST $EPUB_CACHE_CONFIG_DB_PORT; do
      echo "|...|"
      sleep 1
      WAIT=$(($WAIT + 1))
      if [ "$WAIT" -gt 180 ]; then
        echo "Error: Timeout reached for waiting database to start"
        exit 1
      fi
done

cd /go/src/epubcache
$GOPATH/bin/gin  --laddr 0.0.0.0 --port $GIN_PORT --appPort $APP_PORT  run main.go