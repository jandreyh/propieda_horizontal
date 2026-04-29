package entities

import "context"

// SendInput agrupa los datos necesarios para enviar un mensaje por un
// canal. Es la entrada comun que reciben todos los adaptadores concretos.
type SendInput struct {
	RecipientUserID string
	Channel         Channel
	Subject         string
	Body            string
	Payload         []byte
}

// SendResult es el resultado de un intento de envio.
type SendResult struct {
	ProviderMessageID string
	ProviderStatus    string
}

// ChannelSender es la abstraccion de un proveedor de envio para un canal.
// Cada canal (email, push, whatsapp, sms) tendra una implementacion
// concreta. Las implementaciones concretas viven fuera del dominio (infra
// o adaptadores externos). Aqui se define solo la interfaz.
type ChannelSender interface {
	// Send envia un mensaje. Devuelve el resultado del proveedor o un
	// error si el envio fallo.
	Send(ctx context.Context, in SendInput) (SendResult, error)

	// Channel devuelve el canal que este sender implementa.
	Channel() Channel
}
