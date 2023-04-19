# DB

We use a DB to persist execution information.

## Developing Against Postgres

Run a local postgres instance in Docker:

```bash
docker run -p 5432:5432 --name amplify-postgres -e POSTGRES_DB=amplify -e POSTGRES_PASSWORD=mysecretpassword -d postgres
```

The associated connection string would be:

```
postgres://postgres:mysecretpassword@localhost/amplify?sslmode=disable
```