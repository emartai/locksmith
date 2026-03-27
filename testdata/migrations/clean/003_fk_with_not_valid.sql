CREATE INDEX CONCURRENTLY idx_orders_user_id ON orders(user_id);
ALTER TABLE orders ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;
ALTER TABLE orders VALIDATE CONSTRAINT fk_user;
