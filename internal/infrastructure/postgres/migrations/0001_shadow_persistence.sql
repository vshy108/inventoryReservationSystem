CREATE TABLE inventory_items (
    product_id TEXT PRIMARY KEY,
    total_stock INTEGER NOT NULL CHECK (total_stock >= 0),
    confirmed_count INTEGER NOT NULL CHECK (confirmed_count >= 0),
    active_reservation_count INTEGER NOT NULL CHECK (active_reservation_count >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (total_stock >= confirmed_count + active_reservation_count)
);

CREATE TABLE reservations (
    reservation_id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES inventory_items(product_id),
    user_id TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('Active', 'Confirmed', 'Cancelled', 'Expired')),
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (expires_at > created_at),
    CHECK ((state = 'Confirmed') = (confirmed_at IS NOT NULL)),
    CHECK ((state = 'Cancelled') = (cancelled_at IS NOT NULL)),
    CHECK ((state = 'Expired') = (expired_at IS NOT NULL))
);

CREATE INDEX reservations_product_state_idx ON reservations(product_id, state);
CREATE INDEX reservations_expires_at_idx ON reservations(expires_at) WHERE state = 'Active';

CREATE TABLE idempotency_keys (
    idempotency_key TEXT PRIMARY KEY,
    operation TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    response_hash TEXT NOT NULL,
    reservation_id TEXT REFERENCES reservations(reservation_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    CHECK (expires_at > created_at)
);

CREATE INDEX idempotency_keys_expires_at_idx ON idempotency_keys(expires_at);

CREATE TABLE outbox_events (
    outbox_id BIGSERIAL PRIMARY KEY,
    event_id TEXT NOT NULL UNIQUE,
    event_type TEXT NOT NULL CHECK (event_type IN ('Reserved', 'ReserveRejected', 'Confirmed', 'Cancelled', 'Expired')),
    product_id TEXT NOT NULL,
    reservation_id TEXT REFERENCES reservations(reservation_id),
    user_id TEXT,
    reason TEXT,
    occurred_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    observe_only BOOLEAN NOT NULL DEFAULT true,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX outbox_events_unpublished_idx ON outbox_events(created_at, outbox_id) WHERE published_at IS NULL;
CREATE INDEX outbox_events_product_idx ON outbox_events(product_id, occurred_at);