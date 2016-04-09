# How to use docker-compose on production

```
# docker-compose up -d
# docker-compose logs
```

How to import your bookmark file to gobookmark :

* place your bookmark file in imports folder, next :

```
# docker-compose stop gobookmark
# docker-compose run --rm gobookmark ./gobookmark import --reset /imports/bookmarks_public_20160318_101550.html
# docker-compose up -d gobookmark
```
