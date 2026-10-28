DELETE FROM company_settings
WHERE key IN ('due_reminder_template', 'language_due_reminder_template');

DROP TABLE IF EXISTS receivable_reminders;
