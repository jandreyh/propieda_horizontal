// Package entities define las entidades de dominio del modulo notifications.
//
// Reglas (CLAUDE.md):
//   - Las entidades NO conocen JSON ni DB (no tags).
//   - El tenant es implicito por la base de datos.
package entities

// Channel enumera los canales de notificacion validos.
type Channel string

const (
	// ChannelEmail canal de correo electronico.
	ChannelEmail Channel = "email"
	// ChannelPush canal de notificaciones push (FCM/APNs).
	ChannelPush Channel = "push"
	// ChannelWhatsApp canal de WhatsApp Business API.
	ChannelWhatsApp Channel = "whatsapp"
	// ChannelSMS canal de mensajes SMS.
	ChannelSMS Channel = "sms"
)

// IsValid indica si el canal es uno de los enumerados.
func (c Channel) IsValid() bool {
	switch c {
	case ChannelEmail, ChannelPush, ChannelWhatsApp, ChannelSMS:
		return true
	}
	return false
}

// RequiresConsent indica si el canal requiere consentimiento explicito
// antes de enviar notificaciones (WhatsApp y SMS).
func (c Channel) RequiresConsent() bool {
	return c == ChannelWhatsApp || c == ChannelSMS
}

// Platform enumera las plataformas de push tokens.
type Platform string

const (
	// PlatformIOS plataforma Apple iOS (APNs).
	PlatformIOS Platform = "ios"
	// PlatformAndroid plataforma Google Android (FCM).
	PlatformAndroid Platform = "android"
	// PlatformWeb plataforma Web (Web Push).
	PlatformWeb Platform = "web"
)

// IsValid indica si la plataforma es una de las enumeradas.
func (p Platform) IsValid() bool {
	switch p {
	case PlatformIOS, PlatformAndroid, PlatformWeb:
		return true
	}
	return false
}
