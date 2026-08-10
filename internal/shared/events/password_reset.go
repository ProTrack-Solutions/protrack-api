package events

// PasswordResetEmail é publicado na exchange "protrack.ex.eventos" com a
// routing key "email.password_reset" e consumido pelo email_consumer para
// disparar o e-mail de recuperação de senha de forma assíncrona.
type PasswordResetEmail struct {
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	ResetURL string `json:"reset_url"`
}

// PasswordChangedEmail é publicado após uma troca de senha bem-sucedida,
// como notificação de segurança (routing key "email.password_changed").
type PasswordChangedEmail struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}
