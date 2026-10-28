-- Lembretes de cobrança enviados manualmente pelo WhatsApp a partir de uma
-- parcela do contas a receber. Guarda o histórico e trava reenvio em 24h.
CREATE TABLE IF NOT EXISTS receivable_reminders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    account_receivable_id UUID NOT NULL REFERENCES accounts_receivable(id) ON DELETE CASCADE,
    whatsapp_message_id UUID NULL REFERENCES whatsapp_messages(id) ON DELETE SET NULL,
    template_name VARCHAR(128) NOT NULL,
    recipient_phone VARCHAR(20) NOT NULL,
    sent_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_receivable_reminders_account ON receivable_reminders (account_receivable_id, sent_at DESC);

-- Template do lembrete de parcela a vencer. Começa vazio: a empresa só passa a
-- enviar lembrete de conta a vencer depois de configurar um template aprovado.
-- Contas vencidas continuam usando o sale_overdue_template.
INSERT INTO company_settings (company_id, key, value)
SELECT c.id, 'due_reminder_template', '""'::jsonb
FROM companies c
WHERE NOT EXISTS (
    SELECT 1
    FROM company_settings cs
    WHERE cs.company_id = c.id
      AND cs.key = 'due_reminder_template'
);

INSERT INTO company_settings (company_id, key, value)
SELECT c.id, 'language_due_reminder_template', '"pt_BR"'::jsonb
FROM companies c
WHERE NOT EXISTS (
    SELECT 1
    FROM company_settings cs
    WHERE cs.company_id = c.id
      AND cs.key = 'language_due_reminder_template'
);
