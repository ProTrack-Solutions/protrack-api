CREATE TABLE whatsapp_bot_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    instance_name VARCHAR(100) NOT NULL, -- nome da instância na Evolution API
    welcome_message TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id),
    UNIQUE (instance_name)
);

CREATE TABLE whatsapp_bot_menu_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_config_id UUID NOT NULL REFERENCES whatsapp_bot_configs(id) ON DELETE CASCADE,
    option_key VARCHAR(10) NOT NULL,       -- "1", "2", "3"
    label VARCHAR(100) NOT NULL,           -- "Ver cardápio" (exibido na welcome_message)
    response_message TEXT NOT NULL,        -- o que o bot manda quando o cliente escolhe essa opção
    order_index INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bot_config_id, option_key)
);

CREATE INDEX idx_whatsapp_bot_menu_options_bot_config_id ON whatsapp_bot_menu_options(bot_config_id);