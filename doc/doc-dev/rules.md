# Locksmith — Rules Reference

## Overview

Locksmith MVP ships with 10 rules covering the most common causes of Postgres migration incidents. Each rule includes: the detection logic, the danger explanation, the lock type acquired, and the correct safe rewrite.

Rules are identified by a `SCREAMING_SNAKE_CASE` ID. This ID is used in ignore comments and config files.

---

## Rule Severity Levels

| Level | Meaning | Default Exit Code |
|-------|---------|-------------------|
| `DANGEROUS` | Will cause downtime or data loss on tables of any significant size | 1 (blocks CI) |
| `WARNING` | Safe on small tables, dangerous at scale, or depends on context | 2 (configurable) |
| `INFO` | Best practice suggestion, no lock risk | 0 |

---

## Rule 1 — ADD_COLUMN_DEFAULT

**Severity**: DANGEROUS  
**Rule ID**: `ADD_COLUMN_DEFAULT`

### What it detects
```sql
ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
```
Any `ADD COLUMN` with a `NOT NULL` constraint and a `DEFAULT` value on Postgres 10 or earlier.

### Why it's dangerous
On Postgres 10 and earlier, adding a column with a `NOT NULL DEFAULT` rewrites the **entire table** — every row is updated to include the new column value. For a 10M-row table this can take minutes. During this time an `ACCESS EXCLUSIVE` lock is held, blocking all reads and writes.

On Postgres 11+, this specific case was optimized and is safe. However, Locksmith flags it by default because many production systems still run mixed or older versions, and the config should explicitly acknowledge the Postgres version.

### Lock type
`ACCESS EXCLUSIVE` — blocks all reads and writes

### Safe rewrite
```sql
-- Step 1: Add the column as nullable (instant on all versions)
ALTER TABLE users ADD COLUMN status TEXT;

-- Step 2: Backfill in batches (outside a transaction, with delays)
UPDATE users SET status = 'active' WHERE id BETWEEN 1 AND 10000;
-- ... repeat in batches

-- Step 3: Add NOT NULL constraint (only after backfill is complete)
ALTER TABLE users ALTER COLUMN status SET NOT NULL;
```

### Ignore
```sql
-- locksmith:ignore ADD_COLUMN_DEFAULT
ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
```

---

## Rule 2 — ALTER_COLUMN_TYPE

**Severity**: DANGEROUS  
**Rule ID**: `ALTER_COLUMN_TYPE`

### What it detects
```sql
ALTER TABLE users ALTER COLUMN age TYPE BIGINT;
ALTER TABLE orders ALTER COLUMN total TYPE NUMERIC(12,4);
```
Any `ALTER COLUMN ... TYPE` change that is not a safe implicit cast.

### Why it's dangerous
Changing a column's type requires Postgres to rewrite every row in the table to validate and convert the existing data. This acquires an `ACCESS EXCLUSIVE` lock for the full duration of the rewrite.

Safe casts (e.g., `VARCHAR(50)` → `VARCHAR(100)`, `INT` → `BIGINT` via `USING`) are not always optimized away.

### Lock type
`ACCESS EXCLUSIVE` — blocks all reads and writes

### Safe rewrite
```sql
-- Option 1: Add a new column, backfill, swap
ALTER TABLE users ADD COLUMN age_new BIGINT;
UPDATE users SET age_new = age::BIGINT;
ALTER TABLE users ALTER COLUMN age_new SET NOT NULL;
-- After deploy: rename and drop old column in a separate migration

-- Option 2: Use a CHECK constraint instead of type enforcement
ALTER TABLE users ADD CONSTRAINT age_is_integer CHECK (age = floor(age));
```

---

## Rule 3 — NOT_NULL_NO_DEFAULT

**Severity**: DANGEROUS  
**Rule ID**: `NOT_NULL_NO_DEFAULT`

### What it detects
```sql
ALTER TABLE users ALTER COLUMN email SET NOT NULL;
```
Adding a `NOT NULL` constraint to an existing column without a `DEFAULT` and without `NOT VALID`.

### Why it's dangerous
Postgres must scan the entire table to verify no existing rows violate the constraint. This scan holds a full table lock.

### Lock type
`ACCESS EXCLUSIVE` — blocks all reads and writes during the constraint check scan

### Safe rewrite
```sql
-- Step 1: Add a CHECK constraint with NOT VALID (skips existing rows)
ALTER TABLE users ADD CONSTRAINT email_not_null CHECK (email IS NOT NULL) NOT VALID;

-- Step 2: Validate in a separate transaction (uses lower lock level)
ALTER TABLE users VALIDATE CONSTRAINT email_not_null;

-- Step 3: Only then set NOT NULL if needed for ORM compatibility
-- (optional — the CHECK constraint is equivalent for query purposes)
```

---

## Rule 4 — INDEX_WITHOUT_CONCURRENTLY

**Severity**: DANGEROUS  
**Rule ID**: `INDEX_WITHOUT_CONCURRENTLY`

### What it detects
```sql
CREATE INDEX idx_users_email ON users(email);
CREATE UNIQUE INDEX idx_users_username ON users(username);
```
Any `CREATE INDEX` without the `CONCURRENTLY` keyword, on any table.

### Why it's dangerous
A standard `CREATE INDEX` holds a `SHARE` lock on the table for the full duration of the index build. On a large table this can block writes for minutes. `CONCURRENTLY` builds the index without holding a lock (takes longer but never blocks).

### Lock type
`SHARE` — blocks writes, allows reads

### Safe rewrite
```sql
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
CREATE UNIQUE INDEX CONCURRENTLY idx_users_username ON users(username);
```

### Caveat
`CREATE INDEX CONCURRENTLY` cannot run inside a transaction block. If your migration framework wraps everything in a transaction, you need to disable that for this migration. In Rails: `disable_ddl_transaction!`. In Flyway: `outOfOrder=true`.

---

## Rule 5 — DROP_OBJECT

**Severity**: DANGEROUS  
**Rule ID**: `DROP_OBJECT`

### What it detects
```sql
DROP TABLE users;
DROP TABLE IF EXISTS old_sessions;
ALTER TABLE orders DROP COLUMN legacy_status;
ALTER TABLE orders DROP COLUMN IF EXISTS legacy_status;
```
Any `DROP TABLE` or `DROP COLUMN` statement.

### Why it's dangerous
Two separate risks:

1. **Lock risk**: `DROP TABLE` and `DROP COLUMN` acquire `ACCESS EXCLUSIVE` locks
2. **Data loss risk**: If any running application code still references the dropped column or table, it will immediately start throwing errors. Zero-downtime deployments require removing column references from application code **before** dropping the column from the database.

### Safe rewrite (Blue-Green Column Removal)
```sql
-- Migration 1 (deploy with app code that no longer reads the column):
-- Do nothing to the DB yet. Just deploy the app change.

-- Migration 2 (after confirming no app code reads the column):
ALTER TABLE orders DROP COLUMN IF EXISTS legacy_status;
```

For table drops, ensure all foreign key references are removed first and the table is not referenced by any active queries.

---

## Rule 6 — FOREIGN_KEY_NO_NOT_VALID

**Severity**: DANGEROUS  
**Rule ID**: `FOREIGN_KEY_NO_NOT_VALID`

### What it detects
```sql
ALTER TABLE orders ADD CONSTRAINT fk_orders_users 
  FOREIGN KEY (user_id) REFERENCES users(id);
```
Any `ADD CONSTRAINT ... FOREIGN KEY` without the `NOT VALID` clause.

### Why it's dangerous
Without `NOT VALID`, Postgres validates the foreign key constraint against every existing row in the table immediately. On a large `orders` table this scan holds a `SHARE ROW EXCLUSIVE` lock that blocks all writes for the duration.

### Lock type
`SHARE ROW EXCLUSIVE` — blocks inserts, updates, deletes

### Safe rewrite
```sql
-- Step 1: Add the constraint with NOT VALID (skips existing rows, instant lock release)
ALTER TABLE orders ADD CONSTRAINT fk_orders_users 
  FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;

-- Step 2: Validate in a separate migration (uses SHARE UPDATE EXCLUSIVE — lower impact)
ALTER TABLE orders VALIDATE CONSTRAINT fk_orders_users;
```

---

## Rule 7 — TRUNCATE_TABLE

**Severity**: DANGEROUS  
**Rule ID**: `TRUNCATE_TABLE`

### What it detects
```sql
TRUNCATE users;
TRUNCATE users CASCADE;
TRUNCATE sessions, users;
```
Any `TRUNCATE` statement.

### Why it's dangerous
`TRUNCATE` acquires an `ACCESS EXCLUSIVE` lock on the target table. If the table is referenced by foreign keys in other tables, a `CASCADE` truncate can delete data across multiple tables simultaneously. Even without `CASCADE`, the lock blocks all access.

Additionally, `TRUNCATE` is not MVCC-safe in all contexts — it can conflict with ongoing transactions in unexpected ways.

### Lock type
`ACCESS EXCLUSIVE` — blocks all reads and writes

### Safe alternative
For test data cleanup: use `DELETE FROM table WHERE condition` in batches.
For full table clearing: schedule during a maintenance window with explicit acknowledgment.

---

## Rule 8 — MISSING_FK_INDEX

**Severity**: WARNING  
**Rule ID**: `MISSING_FK_INDEX`

### What it detects
```sql
ALTER TABLE orders ADD CONSTRAINT fk_orders_users 
  FOREIGN KEY (user_id) REFERENCES users(id);
-- No corresponding: CREATE INDEX ON orders(user_id)
```
A foreign key constraint added without a corresponding index on the referencing column.

### Why it's dangerous
Without an index on the foreign key column:
- `ON DELETE CASCADE` operations on the parent table will do full table scans on the child table
- Queries joining on the foreign key will be slow
- Lock contention increases during parent table modifications

### Safe pattern
```sql
-- Always create the index before or alongside the FK constraint
CREATE INDEX CONCURRENTLY idx_orders_user_id ON orders(user_id);

ALTER TABLE orders ADD CONSTRAINT fk_orders_users 
  FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;

ALTER TABLE orders VALIDATE CONSTRAINT fk_orders_users;
```

---

## Rule 9 — RENAME_OBJECT

**Severity**: DANGEROUS  
**Rule ID**: `RENAME_OBJECT`

### What it detects
```sql
ALTER TABLE users RENAME COLUMN email TO email_address;
ALTER TABLE users RENAME TO accounts;
```
Any `RENAME COLUMN` or `RENAME TABLE` statement.

### Why it's dangerous
Renaming a column or table is an **instant DDL change** but it **immediately breaks** any application code that references the old name. This includes:
- ORM queries using the old column name
- Raw SQL queries
- Database views referencing the column
- Stored procedures or functions

Even with a zero-downtime deployment, there is a window where old pods still run and will throw errors.

### Safe rewrite (for column rename)
```sql
-- Step 1: Add a new column with the new name
ALTER TABLE users ADD COLUMN email_address TEXT;

-- Step 2: Keep both columns in sync via trigger or dual-write in application

-- Step 3: Backfill the new column
UPDATE users SET email_address = email;

-- Step 4: Deploy application code using the new column name

-- Step 5: Drop the old column in a separate migration after confirming no usage
ALTER TABLE users DROP COLUMN email;
```

---

## Rule 10 — MISSING_LOCK_TIMEOUT

**Severity**: WARNING  
**Rule ID**: `MISSING_LOCK_TIMEOUT`

### What it detects
Any migration containing a `DANGEROUS`-level statement without a preceding `SET lock_timeout` or `SET statement_timeout`.

```sql
-- Flagged: no timeout set
ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active';

-- Not flagged: timeout is set
SET lock_timeout = '5s';
ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
```

### Why it matters
Without a lock timeout, a migration waiting to acquire a lock will wait indefinitely. In a busy production database this means:
- The migration holds a lock wait queue position
- New queries that need the same lock queue behind it
- Within seconds you have a "lock pileup" that effectively takes down the service

Setting `lock_timeout = '2s'` means the migration fails fast instead of cascading into an outage.

### Safe pattern
```sql
-- Always set at the top of any migration with lock-acquiring statements
SET lock_timeout = '2s';
SET statement_timeout = '30s';

-- Then proceed with the migration
ALTER TABLE users ...;
```

---

## Rule Summary Table

| # | Rule ID | Severity | Detects |
|---|---------|----------|---------|
| 1 | `ADD_COLUMN_DEFAULT` | DANGEROUS | ADD COLUMN with NOT NULL + DEFAULT |
| 2 | `ALTER_COLUMN_TYPE` | DANGEROUS | ALTER COLUMN TYPE changes |
| 3 | `NOT_NULL_NO_DEFAULT` | DANGEROUS | SET NOT NULL on existing column |
| 4 | `INDEX_WITHOUT_CONCURRENTLY` | DANGEROUS | CREATE INDEX without CONCURRENTLY |
| 5 | `DROP_OBJECT` | DANGEROUS | DROP TABLE or DROP COLUMN |
| 6 | `FOREIGN_KEY_NO_NOT_VALID` | DANGEROUS | ADD FOREIGN KEY without NOT VALID |
| 7 | `TRUNCATE_TABLE` | DANGEROUS | TRUNCATE on any table |
| 8 | `MISSING_FK_INDEX` | WARNING | FK constraint without index |
| 9 | `RENAME_OBJECT` | DANGEROUS | RENAME COLUMN or RENAME TABLE |
| 10 | `MISSING_LOCK_TIMEOUT` | WARNING | Dangerous op without lock_timeout set |

---

## Ignore Comment Syntax

```sql
-- locksmith:ignore RULE_ID
<statement to ignore>
```

The ignore comment must appear on the line immediately before the statement it applies to.

### Example
```sql
-- locksmith:ignore INDEX_WITHOUT_CONCURRENTLY
CREATE INDEX idx_small_table ON config(key);
```

---

## Config-Based Rule Severity Override

In `locksmith.yml`:
```yaml
rules:
  MISSING_FK_INDEX: error       # Promote to DANGEROUS
  MISSING_LOCK_TIMEOUT: ignore  # Suppress entirely
  DROP_OBJECT: warning          # Demote to WARNING
```
