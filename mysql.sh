# a command to prepare mysql database for use with cache
docker run --name cache-mysql -e MYSQL_ROOT_PASSWORD=1 -e MYSQL_DATABASE=test -p 3306:3306 -d mysql:5.7