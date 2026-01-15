-- Clauser Feature Schema
-- Run against vinagents database

-- Clauser session table
CREATE TABLE app.clausers (
    clauser_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES app.users(user_id) ON DELETE CASCADE,
    title TEXT,
    agreement_a TEXT,
    agreement_b TEXT,
    clause_a TEXT,
    clause_b TEXT,
    agent_run_id UUID REFERENCES app.agent_runs(agent_run_id) ON DELETE SET NULL,
    clause_c_history JSONB NOT NULL DEFAULT '[]'::JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clausers_user_id ON app.clausers(user_id);

-- Clauser outputs table (immutable - no updated_at)
CREATE TABLE app.clauser_outputs (
    clauser_output_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clauser_id UUID NOT NULL REFERENCES app.clausers(clauser_id) ON DELETE CASCADE,
    agent_run_id UUID REFERENCES app.agent_runs(agent_run_id) ON DELETE SET NULL,
    ordinal INTEGER NOT NULL,
    group_name TEXT NOT NULL,
    group_ordinal INTEGER NOT NULL DEFAULT 0,
    title TEXT NOT NULL,
    kind TEXT NOT NULL,
    content JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clauser_outputs_clauser_id ON app.clauser_outputs(clauser_id);

-- Trigger for updated_at on clausers
CREATE TRIGGER update_clausers_updated_at
    BEFORE UPDATE ON app.clausers
    FOR EACH ROW
    EXECUTE FUNCTION app.update_updated_at_column();
