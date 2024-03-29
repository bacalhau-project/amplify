# DB

We use a DB to persist execution information.

## Connecting to the Production Database

Get the Postgres password from the amplify VM:

```
gcloud compute ssh amplify-vm-production-0 -- cat /terraform_node/variables | grep AMPLIFY_DB_URI
```

Then connect to the DB instance and use the password:

```
gcloud sql connect postgres-instance-production-b8a228db --user=postgres --quiet
```

### Listing the Recent Results Metadata For a Result Type

```
\c amplify
SELECT * FROM result_metadata_type ORDER BY id DESC LIMIT 10; -- pick one of the result types
SELECT * FROM result_metadata WHERE type_id = '138495' ORDER BY id DESC LIMIT 10;
```

### Deleting the Result Analytics Data

```
\c amplify
TRUNCATE TABLE result_metadata;
```

## Developing Against Postgres

Run a local postgres instance in Docker:

```bash
docker run -p 5432:5432 --name amplify-postgres -e POSTGRES_DB=amplify -e POSTGRES_PASSWORD=mysecretpassword -d postgres
```

The associated connection string would be:

```
postgres://postgres:mysecretpassword@localhost/amplify?sslmode=disable
```