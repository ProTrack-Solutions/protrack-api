CREATE TYPE announcement_type AS ENUM ('info', 'warning', 'success', 'maintenance');

CREATE TABLE IF NOT EXISTS announcements(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL, -- Corrigido para snake_case para alinhar com o restante
    title VARCHAR(150) NOT NULL,
    content TEXT NOT NULL,
    type announcement_type NOT NULL DEFAULT 'info',
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID, 
    updated_by UUID,
    deleted_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_announcements_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE -- Vírgula removida daqui
);

CREATE INDEX idx_announcements_visibility ON announcements (is_active, starts_at, expires_at);