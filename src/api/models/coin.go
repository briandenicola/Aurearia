package models

import "time"

type Material string

const (
	MaterialGold     Material = "Gold"
	MaterialSilver   Material = "Silver"
	MaterialBronze   Material = "Bronze"
	MaterialCopper   Material = "Copper"
	MaterialElectrum Material = "Electrum"
	MaterialOther    Material = "Other"
)

type Category string

const (
	CategoryRoman     Category = "Roman"
	CategoryGreek     Category = "Greek"
	CategoryByzantine Category = "Byzantine"
	CategoryModern    Category = "Modern"
	CategoryOther     Category = "Other"
)

type ImageType string

const (
	ImageTypeObverse ImageType = "obverse"
	ImageTypeReverse ImageType = "reverse"
	ImageTypeDetail  ImageType = "detail"
	ImageTypeOther   ImageType = "other"
)

type Coin struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	Name                string     `gorm:"not null" json:"name" binding:"max=200"`
	Category            Category   `gorm:"type:varchar(20);not null;default:'Other'" json:"category"`
	Denomination        string     `json:"denomination" binding:"max=200"`
	Ruler               string     `json:"ruler" binding:"max=200"`
	Era                 string     `json:"era" binding:"max=200"`
	Mint                string     `json:"mint" binding:"max=200"`
	Material            Material   `gorm:"type:varchar(20);default:'Other'" json:"material"`
	WeightGrams         *float64   `json:"weightGrams"`
	DiameterMm          *float64   `json:"diameterMm"`
	Grade               string     `json:"grade" binding:"max=100"`
	ObverseInscription  string     `json:"obverseInscription" binding:"max=1000"`
	ReverseInscription  string     `json:"reverseInscription" binding:"max=1000"`
	ObverseDescription  string     `json:"obverseDescription" binding:"max=2000"`
	ReverseDescription  string     `json:"reverseDescription" binding:"max=2000"`
	RarityRating        string     `json:"rarityRating" binding:"max=100"`
	PurchasePrice       *float64   `json:"purchasePrice"`
	CurrentValue        *float64   `json:"currentValue"`
	PurchaseDate        *time.Time `json:"purchaseDate"`
	PurchaseLocation    string     `json:"purchaseLocation" binding:"max=500"`
	Notes               string     `gorm:"type:text" json:"notes" binding:"max=5000"`
	AIAnalysis          string     `gorm:"type:text;column:ai_analysis" json:"aiAnalysis"`
	ObverseAnalysis     string     `gorm:"type:text;column:obverse_analysis" json:"obverseAnalysis"`
	ReverseAnalysis     string     `gorm:"type:text;column:reverse_analysis" json:"reverseAnalysis"`
	ReferenceURL        string     `json:"referenceUrl" binding:"max=2000"`
	ReferenceText       string     `json:"referenceText" binding:"max=2000"`
	IsWishlist          bool       `gorm:"default:false" json:"isWishlist"`
	IsSold              bool       `gorm:"default:false" json:"isSold"`
	SoldPrice           *float64   `json:"soldPrice"`
	SoldDate            *time.Time `json:"soldDate"`
	SoldTo              string     `json:"soldTo"`
	ListingStatus       string     `gorm:"type:varchar(20);default:''" json:"listingStatus"`
	ListingCheckedAt    *time.Time `json:"listingCheckedAt"`
	ListingCheckReason  string     `gorm:"type:text" json:"listingCheckReason"`
	IsPrivate           bool       `gorm:"default:false" json:"isPrivate"`
	UserID              uint       `gorm:"not null" json:"userId"`
	User                User       `gorm:"foreignKey:UserID" json:"-"`
	Images              []CoinImage `gorm:"foreignKey:CoinID" json:"images"`
	Tags                []Tag       `gorm:"many2many:coin_tags" json:"tags"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type CoinImage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CoinID    uint      `gorm:"not null" json:"coinId"`
	FilePath  string    `gorm:"not null" json:"filePath"`
	ImageType ImageType `gorm:"type:varchar(20);default:'other'" json:"imageType"`
	IsPrimary bool      `gorm:"default:false" json:"isPrimary"`
	CreatedAt time.Time `json:"createdAt"`
}
