package models

import "time"

type ShipmentCarrier string

const (
	ShipmentCarrierUSPS  ShipmentCarrier = "usps"
	ShipmentCarrierUPS   ShipmentCarrier = "ups"
	ShipmentCarrierFedEx ShipmentCarrier = "fedex"
	ShipmentCarrierOther ShipmentCarrier = "other"
)

type ShipmentStatus string

const (
	ShipmentStatusPending        ShipmentStatus = "pending"
	ShipmentStatusLabelCreated   ShipmentStatus = "label_created"
	ShipmentStatusInTransit      ShipmentStatus = "in_transit"
	ShipmentStatusOutForDelivery ShipmentStatus = "out_for_delivery"
	ShipmentStatusDelivered      ShipmentStatus = "delivered"
	ShipmentStatusException      ShipmentStatus = "exception"
	ShipmentStatusReturned       ShipmentStatus = "returned"
	ShipmentStatusUnknown        ShipmentStatus = "unknown"
)

type ShipmentStatusSource string

const (
	ShipmentStatusSourceManual ShipmentStatusSource = "manual"
	ShipmentStatusSourceAPI    ShipmentStatusSource = "carrier_api"
)

type Shipment struct {
	ID                      uint                 `gorm:"primaryKey" json:"id"`
	UserID                  uint                 `gorm:"not null;index;uniqueIndex:idx_shipments_user_coin" json:"userId"`
	User                    User                 `gorm:"foreignKey:UserID" json:"-"`
	CoinID                  uint                 `gorm:"not null;index;uniqueIndex:idx_shipments_user_coin" json:"coinId"`
	Coin                    Coin                 `gorm:"foreignKey:CoinID" json:"-"`
	Carrier                 ShipmentCarrier      `gorm:"type:varchar(20);not null;index" json:"carrier"`
	ManualCarrierName       string               `gorm:"size:100" json:"manualCarrierName"`
	TrackingNumber          string               `gorm:"not null;size:120;index" json:"trackingNumber"`
	CurrentStatus           ShipmentStatus       `gorm:"type:varchar(30);not null;default:'pending';index" json:"currentStatus"`
	CurrentStatusSource     ShipmentStatusSource `gorm:"type:varchar(20);not null;default:'manual'" json:"currentStatusSource"`
	Notes                   string               `gorm:"type:text" json:"notes"`
	ManualOverrideEnabled   bool                 `gorm:"not null;default:false" json:"manualOverrideEnabled"`
	ManualOverrideStatus    ShipmentStatus       `gorm:"type:varchar(30)" json:"manualOverrideStatus"`
	ManualOverrideNote      string               `gorm:"type:text" json:"manualOverrideNote"`
	ManualOverrideUpdatedAt *time.Time           `json:"manualOverrideUpdatedAt"`
	LastSyncedAt            *time.Time           `json:"lastSyncedAt"`
	LastSyncError           string               `gorm:"type:text" json:"lastSyncError"`
	EstimatedDeliveryAt     *time.Time           `gorm:"index" json:"estimatedDeliveryAt"`
	DeliveredAt             *time.Time           `json:"deliveredAt"`
	Events                  []ShipmentEvent      `gorm:"foreignKey:ShipmentID" json:"events,omitempty"`
	CreatedAt               time.Time            `json:"createdAt"`
	UpdatedAt               time.Time            `json:"updatedAt"`
}

type ShipmentEvent struct {
	ID           uint                 `gorm:"primaryKey" json:"id"`
	ShipmentID   uint                 `gorm:"not null;index;uniqueIndex:idx_shipment_events_unique_key" json:"shipmentId"`
	Shipment     Shipment             `gorm:"foreignKey:ShipmentID" json:"-"`
	UserID       uint                 `gorm:"not null;index" json:"userId"`
	User         User                 `gorm:"foreignKey:UserID" json:"-"`
	EventKey     string               `gorm:"size:160;not null;uniqueIndex:idx_shipment_events_unique_key" json:"eventKey"`
	Status       ShipmentStatus       `gorm:"type:varchar(30);not null;index" json:"status"`
	StatusSource ShipmentStatusSource `gorm:"type:varchar(20);not null;default:'carrier_api'" json:"statusSource"`
	OccurredAt   time.Time            `gorm:"not null;index" json:"occurredAt"`
	Location     string               `gorm:"size:200" json:"location"`
	Description  string               `gorm:"type:text" json:"description"`
	RawStatus    string               `gorm:"size:120" json:"rawStatus"`
	RawPayload   string               `gorm:"type:text" json:"rawPayload"`
	CreatedAt    time.Time            `json:"createdAt"`
}
