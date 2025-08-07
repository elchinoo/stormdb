-- 001_create_tpcc_schema.up.sql
-- Create seed tables
CREATE TABLE IF NOT EXISTS countries (
    country_id SERIAL PRIMARY KEY,
    country_code CHAR(2) NOT NULL UNIQUE,
    country_name VARCHAR(100) NOT NULL
);
CREATE TABLE IF NOT EXISTS cities (
    city_id SERIAL PRIMARY KEY,
    country_id INT NOT NULL REFERENCES countries(country_id),
    city_name VARCHAR(100) NOT NULL
);

-- Create core TPC-C tables
-- Seed table: warehouse with address data
CREATE TABLE IF NOT EXISTS warehouse (
    w_id SERIAL PRIMARY KEY,
    w_name       VARCHAR(20) NOT NULL,
    w_street_1   VARCHAR(20) NOT NULL,
    w_city       VARCHAR(20) NOT NULL,
    w_state      CHAR(2)    NOT NULL,
    w_zip        CHAR(9)    NOT NULL,
    w_ytd        DECIMAL(12,2) NOT NULL,
    w_tax        DECIMAL(4,4) NOT NULL DEFAULT 0.0000
);
-- Seed table: district with address data and default next order ID
CREATE TABLE IF NOT EXISTS district (
    d_w_id        INT         NOT NULL,
    d_id          INT         NOT NULL,
    d_name        VARCHAR(20) NOT NULL,
    d_street_1    VARCHAR(20) NOT NULL,
    d_city        VARCHAR(20) NOT NULL,
    d_state       CHAR(2)     NOT NULL,
    d_zip         CHAR(9)     NOT NULL,
    d_next_o_id   INT         NOT NULL DEFAULT 3001,
    d_ytd         DECIMAL(12,2) DEFAULT 0,
    PRIMARY KEY (d_w_id, d_id),
    FOREIGN KEY (d_w_id) REFERENCES warehouse(w_id)
);
CREATE TABLE IF NOT EXISTS customer (
    c_w_id INT NOT NULL,
    c_d_id INT NOT NULL,
    c_id INT NOT NULL,
    c_balance DECIMAL(12,2) NOT NULL,
    c_ytd_payment DECIMAL(12,2) DEFAULT 0,
    c_payment_cnt INT DEFAULT 0,
    c_delivery_cnt INT DEFAULT 0,
    PRIMARY KEY (c_w_id, c_d_id, c_id),
    FOREIGN KEY (c_w_id,c_d_id) REFERENCES district(d_w_id,d_id)
);

-- Create item and stock tables for TPC-C dynamic data
CREATE TABLE IF NOT EXISTS item (
    i_id     INT PRIMARY KEY,
    i_name   VARCHAR(50) NOT NULL,
    i_price  DECIMAL(5,2) NOT NULL,
    i_data   TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS stock (
    s_w_id   INT NOT NULL,
    s_i_id   INT NOT NULL,
    s_quantity INT NOT NULL,
    s_dist_01 TEXT NOT NULL,
    PRIMARY KEY (s_w_id, s_i_id),
    FOREIGN KEY (s_w_id) REFERENCES warehouse(w_id),
    FOREIGN KEY (s_i_id) REFERENCES item(i_id)
);

-- Supplier-Reorder extension tables
CREATE TABLE IF NOT EXISTS supplier (
    su_w_id INT NOT NULL,
    su_i_id INT NOT NULL,
    su_quantity INT NOT NULL,
    PRIMARY KEY (su_w_id, su_i_id)
);
CREATE TABLE IF NOT EXISTS purchase_order (
    po_id SERIAL PRIMARY KEY,
    po_w_id INT NOT NULL,
    po_i_id INT NOT NULL,
    po_supp_w_id INT NOT NULL,
    po_qty INT NOT NULL,
    po_order_date TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS goods_receipt (
    gr_id SERIAL PRIMARY KEY,
    gr_po_id INT NOT NULL REFERENCES purchase_order(po_id),
    gr_receipt_date TIMESTAMP NOT NULL
);

-- Indices
CREATE INDEX IF NOT EXISTS idx_district_w ON district(d_w_id);
CREATE INDEX IF NOT EXISTS idx_customer_balance ON customer(c_balance);
