package internal

type (
	// {"carid":"test-car1","eventTime":1683137969,"elev":"0.0","lat":39.79323,"long":-86.23885}

	// {"previousZoneId":"zone-north","nextZoneId":null,"carId":"test-car1","vin":null}
	ZoneChangeEvent struct {
		PreviousZoneID string `json:"previousZoneId,omitempty"`
		NextZoneID     string `json:"nextZoneId,omitempty"`
		CarID          string `json:"carId,omitempty"`
		VIN            string `json:"vin,omitempty"`
	}
)

func (evt *ZoneChangeEvent) String() string {
	return "*** ZoneChangeEvent"
}
