package dto

// RegisterPushDeviceRequest es el body de POST /me/push-devices.
type RegisterPushDeviceRequest struct {
	DeviceToken string  `json:"device_token"`
	Platform    string  `json:"platform"` // ios | android | web
	DeviceLabel *string `json:"device_label,omitempty"`
}

// RegisterPushDeviceResponse confirma el registro y devuelve el id que el
// cliente debe persistir para revocarlo cuando cierre sesion.
type RegisterPushDeviceResponse struct {
	ID          string  `json:"id"`
	DeviceToken string  `json:"device_token"`
	Platform    string  `json:"platform"`
	DeviceLabel *string `json:"device_label,omitempty"`
}
