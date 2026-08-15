package domain

type WebhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		Id      string `json:"id"`
		Changes []struct {
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberId      string `json:"phone_number_id"`
				} `json:"metadata"`
				Contacts []struct {
					WaID   string `json:"wa_id"`
					UserId string `json:"user_id"`
				} `json:"contacts"`
				Statuses []struct {
					Id          string `json:"id"`
					Status      string `json:"status"`
					Timestamp   string `json:"timestamp"`
					RecipientId string `json:"recipient_id"`
					Errors      []struct {
						Code      int    `json:"code"`
						Title     string `json:"title"`
						Message   string `json:"message"`
						ErrorData any    `json:"error_data"`
					} `json:"errors"`
				} `json:"statuses"`
			} `json:"value"`
			Field string `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}
