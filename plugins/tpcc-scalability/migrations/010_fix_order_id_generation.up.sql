-- 010_fix_order_id_generation.up.sql
-- Add atomic sequence-based order ID generation to prevent race conditions

-- Create sequences for each warehouse-district combination to ensure unique order IDs
-- This prevents the race condition where multiple workers read the same d_next_o_id

-- First, let's create a sequence for order IDs per warehouse
-- We'll use the existing d_next_o_id values as starting points
DO $$
DECLARE
    warehouse_rec RECORD;
    district_rec RECORD;
    seq_name TEXT;
    current_max_id INT;
BEGIN
    -- For each warehouse
    FOR warehouse_rec IN SELECT w_id FROM warehouse ORDER BY w_id
    LOOP
        -- For each district in the warehouse
        FOR district_rec IN 
            SELECT d_w_id, d_id, d_next_o_id 
            FROM district 
            WHERE d_w_id = warehouse_rec.w_id 
            ORDER BY d_id
        LOOP
            -- Create a sequence for this warehouse-district combination
            seq_name := 'order_id_seq_w' || district_rec.d_w_id || '_d' || district_rec.d_id;
            
            -- Check current maximum order ID in the order table
            EXECUTE format('SELECT COALESCE(MAX(o_id), 0) FROM "order" WHERE o_w_id = %L AND o_d_id = %L', 
                          district_rec.d_w_id, district_rec.d_id) INTO current_max_id;
            
            -- Use the higher of d_next_o_id and current_max_id + 1
            current_max_id := GREATEST(district_rec.d_next_o_id, current_max_id + 1);
            
            -- Drop sequence if it exists
            EXECUTE format('DROP SEQUENCE IF EXISTS %I', seq_name);
            
            -- Create sequence starting from the correct value
            EXECUTE format('CREATE SEQUENCE %I START WITH %s INCREMENT BY 1', seq_name, current_max_id);
            
            RAISE NOTICE 'Created sequence % starting at %', seq_name, current_max_id;
        END LOOP;
    END LOOP;
END
$$;

-- Create a function to get the next order ID atomically
CREATE OR REPLACE FUNCTION get_next_order_id(warehouse_id INT, district_id INT)
RETURNS INT
LANGUAGE plpgsql
AS $$
DECLARE
    seq_name TEXT;
    next_id INT;
BEGIN
    seq_name := 'order_id_seq_w' || warehouse_id || '_d' || district_id;
    EXECUTE format('SELECT nextval(%L)', seq_name) INTO next_id;
    
    -- Update the district table to keep d_next_o_id in sync (for compatibility)
    UPDATE district 
    SET d_next_o_id = next_id + 1 
    WHERE d_w_id = warehouse_id AND d_id = district_id;
    
    RETURN next_id;
END
$$;

-- Add indexes to improve performance
CREATE INDEX IF NOT EXISTS idx_order_w_d_id ON "order"(o_w_id, o_d_id, o_id);
CREATE INDEX IF NOT EXISTS idx_district_w_d ON district(d_w_id, d_id);

-- Add comment explaining the solution
COMMENT ON FUNCTION get_next_order_id(INT, INT) IS 
'Atomically generates unique order IDs per warehouse-district combination using sequences to prevent race conditions in concurrent TPC-C workloads';
