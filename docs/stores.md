# Stores

MySQL not supported due to: no `RETURNING` clause, different escape approaches and other similar uncommon behaviour.

Storage can be picked by

    --storage=[database|files|dump]      Storage type (default: database) [$STORAGE]

## Database

Requires

    --db.dialect=[postgres|sqlite3] SQL dialect (default: sqlite3) [$DB_DIALECT]
    --db.url=                       Database URL (default: file://form.sqlite) [$DB_URL]

Optional

    --db.migrations=                Migrations dir (default: migrations) [$DB_MIGRATIONS]
    --db.migrate                    Apply migration on start [$DB_MIGRATE]

By default, migration is disabled in CLI mode and enabled in [Docker](docker.md) mode.

### Postgres

Supported all major types and arrays of text (`TEXT[]`).

Enums are not supported.

**Example environment:**

```
STORAGE=database
DB_DIALECT=postgres
DB_URL=posgres://postgres:postgres@localhost/postgres?sslmod=disable
```

> Postgres by default converts name text case of column names from `CREATE TABLE` statement to lower.
>
> For example: `CREATE TABLE foo (ID INT)` will cause `id` in returned data.

### SQLite

Since SQLite doesn't support arrays, multiselect will be saved as text where selected options merged by `,`. For
example: if user selected option `Foo` and `Bar`, then `Foo,Bar` will be stored in the database.

**Example environment:**

```
STORAGE=database
DB_DIALECT=sqlite3
DB_URL=file://forms.sqlite
```

> SQLite keeps case of column names from `CREATE TABLE` statement.
>
> For example: `CREATE TABLE foo (ID INT)` will cause `ID` in returned data.

## Files

Stores each submission as single file in JSON with [ULID](https://github.com/ulid/spec) + `.json` in the directory,
equal to table name. ULID is picked since it's uniq as UUID, but allows sorting by time.

It uses atomic write (temp + rename) and should be safe for multiprocess writing; however, in case of unexpected
termination, the temporary files may stay in file system.

It **DOES NOT** escape table name AT ALL; it's up to user to take care of proper table name.

Result-set contains all source fields plus `ID` (string).

Requires:

    --files.path=                   Root dir for form results (default: results) [$FILES_PATH]

**Example environment:**

```
STORAGE=files
FILES_PATH=results
```

## Dump

> since 0.4.1

Dumps record to STDOUT. Used for debugging or database-less forms.