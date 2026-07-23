-- UP: Add user_id to critical tables for multi-user isolation
-- Existing rows are assigned to the admin user for backward compatibility.

DO $$
DECLARE
    admin_id UUID;
BEGIN
    SELECT id INTO admin_id FROM users WHERE username = 'admin' LIMIT 1;

    -- If no admin user exists yet, skip migration (will be handled on next startup)
    IF admin_id IS NULL THEN
        RAISE NOTICE 'No admin user found, skipping user_id migration';
        RETURN;
    END IF;

    -- accounts table: add user_id
    ALTER TABLE accounts ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;
    UPDATE accounts SET user_id = admin_id WHERE user_id IS NULL;
    ALTER TABLE accounts ALTER COLUMN user_id SET NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id);

    -- strategies table: add user_id
    ALTER TABLE strategies ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;
    UPDATE strategies SET user_id = admin_id WHERE user_id IS NULL;
    ALTER TABLE strategies ALTER COLUMN user_id SET NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_strategies_user_id ON strategies(user_id);

    -- backtest_runs table: add user_id
    ALTER TABLE backtest_runs ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;
    UPDATE backtest_runs SET user_id = admin_id WHERE user_id IS NULL;
    ALTER TABLE backtest_runs ALTER COLUMN user_id SET NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_backtest_runs_user_id ON backtest_runs(user_id);

    -- trade_executions table: add user_id
    ALTER TABLE trade_executions ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;
    UPDATE trade_executions SET user_id = admin_id WHERE user_id IS NULL;
    ALTER TABLE trade_executions ALTER COLUMN user_id SET NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_trade_executions_user_id ON trade_executions(user_id);

    -- credentials table: add user_id
    ALTER TABLE credentials ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;
    UPDATE credentials SET user_id = admin_id WHERE user_id IS NULL;
    ALTER TABLE credentials ALTER COLUMN user_id SET NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_credentials_user_id ON credentials(user_id);

    -- settings table: add user_id (allows per-user settings)
    ALTER TABLE settings ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;
    UPDATE settings SET user_id = admin_id WHERE user_id IS NULL;
    CREATE INDEX IF NOT EXISTS idx_settings_user_id ON settings(user_id);

    -- kill_switch_history: add triggered_by user
    ALTER TABLE kill_switch_history ADD COLUMN IF NOT EXISTS triggered_by UUID REFERENCES users(id);
    CREATE INDEX IF NOT EXISTS idx_kill_switch_history_user ON kill_switch_history(triggered_by);
END $$;
