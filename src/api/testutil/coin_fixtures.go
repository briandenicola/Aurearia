package testutil

import (
	"fmt"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// GoldenCoinFixtureName identifies one deterministic representative coin fixture.
type GoldenCoinFixtureName string

const (
	RomanDenariusCore         GoldenCoinFixtureName = "roman-denarius-core"
	GreekTetradrachmValued    GoldenCoinFixtureName = "greek-tetradrachm-valued"
	ByzantineSolidusSetMember GoldenCoinFixtureName = "byzantine-solidus-set-member"
	WishlistAureusTarget      GoldenCoinFixtureName = "wishlist-aureus-target"
	SoldSestertiusArchive     GoldenCoinFixtureName = "sold-sestertius-archive"
	PrivateProvincialBronze   GoldenCoinFixtureName = "private-provincial-bronze"
	TaggedFollisStorage       GoldenCoinFixtureName = "tagged-follis-storage"
	ImageHeavyDrachm          GoldenCoinFixtureName = "image-heavy-drachm"
	ReferenceRichDenarius     GoldenCoinFixtureName = "reference-rich-denarius"
)

var goldenCoinFixtureNames = []GoldenCoinFixtureName{
	RomanDenariusCore,
	GreekTetradrachmValued,
	ByzantineSolidusSetMember,
	WishlistAureusTarget,
	SoldSestertiusArchive,
	PrivateProvincialBronze,
	TaggedFollisStorage,
	ImageHeavyDrachm,
	ReferenceRichDenarius,
}

// GoldenCoinFixtureNames returns the fixture names in deterministic order.
func GoldenCoinFixtureNames() []GoldenCoinFixtureName {
	return cloneFixtureNames(goldenCoinFixtureNames)
}

type GoldenCoinTrait string

const (
	TraitRoman           GoldenCoinTrait = "roman"
	TraitGreek           GoldenCoinTrait = "greek"
	TraitByzantine       GoldenCoinTrait = "byzantine"
	TraitWishlist        GoldenCoinTrait = "wishlist"
	TraitSold            GoldenCoinTrait = "sold"
	TraitPrivate         GoldenCoinTrait = "private"
	TraitTagged          GoldenCoinTrait = "tagged"
	TraitSetMember       GoldenCoinTrait = "set-member"
	TraitStorageLocation GoldenCoinTrait = "storage-location"
	TraitImageHeavy      GoldenCoinTrait = "image-heavy"
	TraitLegacyCustomEra GoldenCoinTrait = "legacy-custom-era"
	TraitValued          GoldenCoinTrait = "valued"
	TraitReferenceRich   GoldenCoinTrait = "reference-rich"
)

type GoldenCoinFixtureInfo struct {
	Name   GoldenCoinFixtureName
	Traits []GoldenCoinTrait
}

type CoinFixture struct {
	Name   GoldenCoinFixtureName
	Traits []GoldenCoinTrait
	Coin   models.Coin
}

type CoinOption func(*models.Coin)

type PersistedGoldenCollection struct {
	Coins            []models.Coin
	StorageLocations map[string]models.StorageLocation
	Tags             map[string]models.Tag
	Sets             map[string]models.CoinSet
}

const (
	fixtureTrayAName        = "Cabinet Tray A"
	fixtureVaultBoxName     = "Vault Box 2"
	fixturePhotographedName = "Photographed"
	fixtureNeedsResearch    = "Needs Research"
	fixtureTwelveCaesars    = "Twelve Caesars"
	fixtureByzantineGold    = "Byzantine Gold"
)

var fixtureTimestamp = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

func ptrFloat(v float64) *float64    { return &v }
func ptrTime(v time.Time) *time.Time { return &v }

func GoldenCoinFixtureCatalog() []GoldenCoinFixtureInfo {
	catalog := make([]GoldenCoinFixtureInfo, 0, len(goldenCoinFixtureNames))
	for _, name := range goldenCoinFixtureNames {
		catalog = append(catalog, GoldenCoinFixtureInfo{Name: name, Traits: cloneTraits(fixtureTraits[name])})
	}
	return catalog
}

func BuildGoldenCoinFixture(name GoldenCoinFixtureName, userID uint, opts ...CoinOption) (CoinFixture, error) {
	builder, ok := fixtureBuilders[name]
	if !ok {
		return CoinFixture{}, fmt.Errorf("unknown golden coin fixture %q", name)
	}
	coin := builder(userID)
	for _, opt := range opts {
		opt(&coin)
	}
	return CoinFixture{Name: name, Traits: cloneTraits(fixtureTraits[name]), Coin: cloneCoin(coin)}, nil
}

func MustBuildGoldenCoinFixture(name GoldenCoinFixtureName, userID uint, opts ...CoinOption) CoinFixture {
	fixture, err := BuildGoldenCoinFixture(name, userID, opts...)
	if err != nil {
		panic(err)
	}
	return fixture
}

func BuildGoldenCoinFixtures(userID uint) []CoinFixture {
	fixtures := make([]CoinFixture, 0, len(goldenCoinFixtureNames))
	for _, name := range goldenCoinFixtureNames {
		fixtures = append(fixtures, MustBuildGoldenCoinFixture(name, userID))
	}
	return fixtures
}

func BuildRomanDenariusCore(userID uint, opts ...CoinOption) models.Coin {
	return MustBuildGoldenCoinFixture(RomanDenariusCore, userID, opts...).Coin
}

func BuildGreekTetradrachmValued(userID uint, opts ...CoinOption) models.Coin {
	return MustBuildGoldenCoinFixture(GreekTetradrachmValued, userID, opts...).Coin
}

func BuildByzantineSolidusSetMember(userID uint, opts ...CoinOption) models.Coin {
	return MustBuildGoldenCoinFixture(ByzantineSolidusSetMember, userID, opts...).Coin
}

func BuildWishlistAureusTarget(userID uint, opts ...CoinOption) models.Coin {
	return MustBuildGoldenCoinFixture(WishlistAureusTarget, userID, opts...).Coin
}

func BuildSoldSestertiusArchive(userID uint, opts ...CoinOption) models.Coin {
	return MustBuildGoldenCoinFixture(SoldSestertiusArchive, userID, opts...).Coin
}

func BuildPrivateProvincialBronze(userID uint, opts ...CoinOption) models.Coin {
	return MustBuildGoldenCoinFixture(PrivateProvincialBronze, userID, opts...).Coin
}

func BuildTaggedFollisStorage(userID uint, opts ...CoinOption) models.Coin {
	return MustBuildGoldenCoinFixture(TaggedFollisStorage, userID, opts...).Coin
}

func BuildImageHeavyDrachm(userID uint, opts ...CoinOption) models.Coin {
	return MustBuildGoldenCoinFixture(ImageHeavyDrachm, userID, opts...).Coin
}

func BuildReferenceRichDenarius(userID uint, opts ...CoinOption) models.Coin {
	return MustBuildGoldenCoinFixture(ReferenceRichDenarius, userID, opts...).Coin
}

func WithCoinMutation(mutator func(*models.Coin)) CoinOption {
	return func(coin *models.Coin) {
		if mutator != nil {
			mutator(coin)
		}
	}
}

func BuildTestStorageLocations(userID uint) []models.StorageLocation {
	return []models.StorageLocation{
		{UserID: userID, Name: fixtureTrayAName, SortOrder: 1, CreatedAt: fixtureTimestamp, UpdatedAt: fixtureTimestamp},
		{UserID: userID, Name: fixtureVaultBoxName, SortOrder: 2, CreatedAt: fixtureTimestamp, UpdatedAt: fixtureTimestamp},
	}
}

func BuildTestTags(userID uint) []models.Tag {
	return []models.Tag{
		{UserID: userID, Name: fixturePhotographedName, Color: "#c9a84c"},
		{UserID: userID, Name: fixtureNeedsResearch, Color: "#4682b4"},
	}
}

func BuildTestCoinSets(userID uint) []models.CoinSet {
	targetDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	return []models.CoinSet{
		{UserID: userID, Name: fixtureTwelveCaesars, Description: "Representative imperial portrait set", Color: "#c9a84c", Icon: "Crown", SetType: models.CoinSetTypeDefined, TargetCompletionDate: &targetDate, CreatedAt: fixtureTimestamp, UpdatedAt: fixtureTimestamp},
		{UserID: userID, Name: fixtureByzantineGold, Description: "Gold issues for set-membership workflow tests", Color: "#b08d57", Icon: "CircleDot", SetType: models.CoinSetTypeOpen, CreatedAt: fixtureTimestamp, UpdatedAt: fixtureTimestamp},
	}
}

// PersistGoldenCollection seeds the golden fixtures into a caller-managed test DB.
// The DB must already be migrated for User, Coin, CoinImage, CoinReference,
// StorageLocation, Tag, CoinTag, CoinSet, and CoinSetMembership models.
func PersistGoldenCollection(db *gorm.DB, userID uint) (*PersistedGoldenCollection, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	persisted := &PersistedGoldenCollection{
		Coins:            make([]models.Coin, 0, len(goldenCoinFixtureNames)),
		StorageLocations: map[string]models.StorageLocation{},
		Tags:             map[string]models.Tag{},
		Sets:             map[string]models.CoinSet{},
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		for _, location := range BuildTestStorageLocations(userID) {
			if err := tx.Create(&location).Error; err != nil {
				return err
			}
			persisted.StorageLocations[location.Name] = location
		}
		for _, tag := range BuildTestTags(userID) {
			if err := tx.Create(&tag).Error; err != nil {
				return err
			}
			persisted.Tags[tag.Name] = tag
		}
		for _, set := range BuildTestCoinSets(userID) {
			if err := tx.Create(&set).Error; err != nil {
				return err
			}
			persisted.Sets[set.Name] = set
		}

		for _, fixture := range BuildGoldenCoinFixtures(userID) {
			coin := cloneCoin(fixture.Coin)
			images := cloneImages(coin.Images)
			references := cloneReferences(coin.References)
			tags := cloneTags(coin.Tags)
			sets := cloneSets(coin.Sets)

			coin.ID = 0
			coin.Images = nil
			coin.References = nil
			coin.Tags = nil
			coin.Sets = nil
			if coin.StorageLocation != nil {
				location, ok := persisted.StorageLocations[coin.StorageLocation.Name]
				if !ok {
					return fmt.Errorf("fixture %s references unknown storage location %q", fixture.Name, coin.StorageLocation.Name)
				}
				coin.StorageLocationID = &location.ID
				coin.StorageLocation = nil
			}

			if err := tx.Create(&coin).Error; err != nil {
				return err
			}

			for _, image := range images {
				image.ID = 0
				image.CoinID = coin.ID
				if err := tx.Create(&image).Error; err != nil {
					return err
				}
			}
			for _, ref := range references {
				ref.ID = 0
				ref.CoinID = coin.ID
				if err := tx.Create(&ref).Error; err != nil {
					return err
				}
			}
			for _, tag := range tags {
				persistedTag, ok := persisted.Tags[tag.Name]
				if !ok {
					return fmt.Errorf("fixture %s references unknown tag %q", fixture.Name, tag.Name)
				}
				if err := tx.Create(&models.CoinTag{CoinID: coin.ID, TagID: persistedTag.ID}).Error; err != nil {
					return err
				}
			}
			for _, set := range sets {
				persistedSet, ok := persisted.Sets[set.Name]
				if !ok {
					return fmt.Errorf("fixture %s references unknown set %q", fixture.Name, set.Name)
				}
				membership := models.CoinSetMembership{CoinID: coin.ID, SetID: persistedSet.ID, AddedAt: fixtureTimestamp, Notes: "golden fixture"}
				if err := tx.Create(&membership).Error; err != nil {
					return err
				}
			}

			var loaded models.Coin
			if err := tx.Preload("Images").Preload("References").Preload("Tags").Preload("Sets").Preload("StorageLocation").First(&loaded, coin.ID).Error; err != nil {
				return err
			}
			persisted.Coins = append(persisted.Coins, loaded)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return persisted, nil
}

var fixtureTraits = map[GoldenCoinFixtureName][]GoldenCoinTrait{
	RomanDenariusCore:         {TraitRoman},
	GreekTetradrachmValued:    {TraitGreek, TraitValued},
	ByzantineSolidusSetMember: {TraitByzantine, TraitSetMember},
	WishlistAureusTarget:      {TraitRoman, TraitWishlist, TraitValued},
	SoldSestertiusArchive:     {TraitRoman, TraitSold},
	PrivateProvincialBronze:   {TraitRoman, TraitPrivate, TraitLegacyCustomEra},
	TaggedFollisStorage:       {TraitRoman, TraitTagged, TraitStorageLocation},
	ImageHeavyDrachm:          {TraitGreek, TraitImageHeavy},
	ReferenceRichDenarius:     {TraitRoman, TraitReferenceRich, TraitSetMember},
}

var fixtureBuilders = map[GoldenCoinFixtureName]func(uint) models.Coin{
	RomanDenariusCore:         romanDenariusCore,
	GreekTetradrachmValued:    greekTetradrachmValued,
	ByzantineSolidusSetMember: byzantineSolidusSetMember,
	WishlistAureusTarget:      wishlistAureusTarget,
	SoldSestertiusArchive:     soldSestertiusArchive,
	PrivateProvincialBronze:   privateProvincialBronze,
	TaggedFollisStorage:       taggedFollisStorage,
	ImageHeavyDrachm:          imageHeavyDrachm,
	ReferenceRichDenarius:     referenceRichDenarius,
}

func baseCoin(userID uint, name string) models.Coin {
	return models.Coin{
		Name:               name,
		Category:           models.CategoryRoman,
		Denomination:       "Denarius",
		Ruler:              "Trajan",
		Era:                models.EraAncient,
		Mint:               "Rome",
		Material:           models.MaterialSilver,
		WeightGrams:        ptrFloat(3.35),
		DiameterMm:         ptrFloat(18),
		Grade:              "VF",
		ObverseInscription: "IMP TRAIANO AVG GER DAC P M TR P",
		ReverseInscription: "SPQR OPTIMO PRINCIPI",
		ObverseDescription: "Laureate bust right",
		ReverseDescription: "Victory standing left",
		RarityRating:       "Common",
		PurchasePrice:      ptrFloat(180),
		CurrentValue:       ptrFloat(225),
		PurchaseDate:       ptrTime(time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)),
		PurchaseLocation:   "Fixture Dealer",
		Notes:              "Deterministic backend fixture coin.",
		ListingStatus:      "unlisted",
		UserID:             userID,
		Images: []models.CoinImage{
			buildImage("obverse", true),
			buildImage("reverse", false),
		},
		CreatedAt: fixtureTimestamp,
		UpdatedAt: fixtureTimestamp,
	}
}

func romanDenariusCore(userID uint) models.Coin {
	coin := baseCoin(userID, "Trajan Denarius Core")
	coin.ReferenceURL = "https://example.test/coins/roman-denarius-core"
	coin.ReferenceText = "RIC II Trajan 147"
	return coin
}

func greekTetradrachmValued(userID uint) models.Coin {
	coin := baseCoin(userID, "Athens Owl Tetradrachm Valued")
	coin.Category = models.CategoryGreek
	coin.Denomination = "Tetradrachm"
	coin.Ruler = "Athens"
	coin.Mint = "Athens"
	coin.Material = models.MaterialSilver
	coin.WeightGrams = ptrFloat(17.18)
	coin.DiameterMm = ptrFloat(24)
	coin.PurchasePrice = ptrFloat(950)
	coin.CurrentValue = ptrFloat(1250)
	coin.PurchaseDate = ptrTime(time.Date(2023, 9, 10, 0, 0, 0, 0, time.UTC))
	coin.ObverseInscription = ""
	coin.ReverseInscription = "ΑΘΕ"
	coin.ObverseDescription = "Helmeted head of Athena right"
	coin.ReverseDescription = "Owl standing right, olive sprig and crescent"
	return coin
}

func byzantineSolidusSetMember(userID uint) models.Coin {
	coin := baseCoin(userID, "Justinian I Solidus Set Member")
	coin.Category = models.CategoryByzantine
	coin.Denomination = "Solidus"
	coin.Ruler = "Justinian I"
	coin.Mint = "Constantinople"
	coin.Material = models.MaterialGold
	coin.WeightGrams = ptrFloat(4.48)
	coin.DiameterMm = ptrFloat(21)
	coin.PurchasePrice = ptrFloat(700)
	coin.CurrentValue = ptrFloat(875)
	coin.Sets = []models.CoinSet{fixtureSet(userID, fixtureByzantineGold)}
	return coin
}

func wishlistAureusTarget(userID uint) models.Coin {
	coin := baseCoin(userID, "Augustus Aureus Wishlist Target")
	coin.Denomination = "Aureus"
	coin.Ruler = "Augustus"
	coin.Material = models.MaterialGold
	coin.WeightGrams = ptrFloat(7.9)
	coin.DiameterMm = ptrFloat(20)
	coin.PurchasePrice = nil
	coin.CurrentValue = ptrFloat(8500)
	coin.PurchaseDate = nil
	coin.PurchaseLocation = ""
	coin.IsWishlist = true
	coin.Notes = "Wishlist target used for purchase workflow tests."
	return coin
}

func soldSestertiusArchive(userID uint) models.Coin {
	coin := baseCoin(userID, "Hadrian Sestertius Archive")
	coin.Denomination = "Sestertius"
	coin.Ruler = "Hadrian"
	coin.Material = models.MaterialBronze
	coin.WeightGrams = ptrFloat(25.1)
	coin.DiameterMm = ptrFloat(32)
	coin.PurchasePrice = ptrFloat(240)
	coin.CurrentValue = ptrFloat(300)
	coin.IsSold = true
	coin.SoldPrice = ptrFloat(310)
	coin.SoldDate = ptrTime(time.Date(2025, 4, 20, 0, 0, 0, 0, time.UTC))
	coin.SoldTo = "Archive Buyer"
	return coin
}

func privateProvincialBronze(userID uint) models.Coin {
	coin := baseCoin(userID, "Alexandria Provincial Bronze Private")
	coin.Denomination = "Drachm"
	coin.Ruler = "Antoninus Pius"
	coin.Era = models.Era("Roman Provincial Year 12")
	coin.Mint = "Alexandria"
	coin.Material = models.MaterialBronze
	coin.IsPrivate = true
	coin.ReferenceText = "Legacy handwritten tray note"
	coin.Notes = "Private coin with a custom legacy era value."
	return coin
}

func taggedFollisStorage(userID uint) models.Coin {
	coin := baseCoin(userID, "Diocletian Follis Tagged Storage")
	coin.Denomination = "Follis"
	coin.Ruler = "Diocletian"
	coin.Material = models.MaterialBronze
	coin.StorageLocation = &models.StorageLocation{UserID: userID, Name: fixtureTrayAName, SortOrder: 1}
	coin.Tags = []models.Tag{
		{UserID: userID, Name: fixturePhotographedName, Color: "#c9a84c"},
		{UserID: userID, Name: fixtureNeedsResearch, Color: "#4682b4"},
	}
	return coin
}

func imageHeavyDrachm(userID uint) models.Coin {
	coin := baseCoin(userID, "Syracuse Drachm Image Heavy")
	coin.Category = models.CategoryGreek
	coin.Denomination = "Drachm"
	coin.Ruler = "Syracuse"
	coin.Mint = "Syracuse"
	coin.Images = []models.CoinImage{
		buildImage("obverse", true),
		buildImage("reverse", false),
		buildImage("detail", false),
		buildImage("other", false),
	}
	return coin
}

func referenceRichDenarius(userID uint) models.Coin {
	coin := baseCoin(userID, "Vespasian Denarius Reference Rich")
	coin.Ruler = "Vespasian"
	coin.ReferenceURL = "https://example.test/coins/reference-rich-denarius"
	coin.ReferenceText = "RIC II.1 Vespasian 772; BMCRE 161; RSC 554"
	coin.References = []models.CoinReference{
		buildReference("RIC", "II.1", "772"),
		buildReference("BMCRE", "", "161"),
		buildReference("RSC", "", "554"),
	}
	coin.Sets = []models.CoinSet{fixtureSet(userID, fixtureTwelveCaesars)}
	return coin
}

func buildImage(imageType models.ImageType, isPrimary bool) models.CoinImage {
	return models.CoinImage{FilePath: fmt.Sprintf("uploads/test-fixtures/%s.webp", imageType), ImageType: imageType, IsPrimary: isPrimary, CreatedAt: fixtureTimestamp}
}

func buildReference(catalog, volume, number string) models.CoinReference {
	return models.CoinReference{Catalog: catalog, Volume: volume, Number: number, InvoiceNumber: fmt.Sprintf("INV-%s-%s", catalog, number), URI: fmt.Sprintf("https://example.test/catalog/%s/%s", catalog, number), CreatedAt: fixtureTimestamp, UpdatedAt: fixtureTimestamp}
}

func fixtureSet(userID uint, name string) models.CoinSet {
	for _, set := range BuildTestCoinSets(userID) {
		if set.Name == name {
			return set
		}
	}
	return models.CoinSet{UserID: userID, Name: name, SetType: models.CoinSetTypeOpen}
}

func cloneCoin(coin models.Coin) models.Coin {
	coin.WeightGrams = cloneFloatPtr(coin.WeightGrams)
	coin.DiameterMm = cloneFloatPtr(coin.DiameterMm)
	coin.PurchasePrice = cloneFloatPtr(coin.PurchasePrice)
	coin.CurrentValue = cloneFloatPtr(coin.CurrentValue)
	coin.CurrentValueUpdatedAt = cloneTimePtr(coin.CurrentValueUpdatedAt)
	coin.PurchaseDate = cloneTimePtr(coin.PurchaseDate)
	coin.SoldPrice = cloneFloatPtr(coin.SoldPrice)
	coin.SoldDate = cloneTimePtr(coin.SoldDate)
	coin.ListingCheckedAt = cloneTimePtr(coin.ListingCheckedAt)
	coin.StorageLocationID = cloneUintPtr(coin.StorageLocationID)
	if coin.StorageLocation != nil {
		location := *coin.StorageLocation
		coin.StorageLocation = &location
	}
	coin.Images = cloneImages(coin.Images)
	coin.References = cloneReferences(coin.References)
	coin.Tags = cloneTags(coin.Tags)
	coin.Sets = cloneSets(coin.Sets)
	return coin
}

func cloneImages(images []models.CoinImage) []models.CoinImage {
	if images == nil {
		return nil
	}
	cloned := make([]models.CoinImage, len(images))
	copy(cloned, images)
	return cloned
}

func cloneReferences(refs []models.CoinReference) []models.CoinReference {
	if refs == nil {
		return nil
	}
	cloned := make([]models.CoinReference, len(refs))
	copy(cloned, refs)
	return cloned
}

func cloneTags(tags []models.Tag) []models.Tag {
	if tags == nil {
		return nil
	}
	cloned := make([]models.Tag, len(tags))
	copy(cloned, tags)
	return cloned
}

func cloneSets(sets []models.CoinSet) []models.CoinSet {
	if sets == nil {
		return nil
	}
	cloned := make([]models.CoinSet, len(sets))
	copy(cloned, sets)
	return cloned
}

func cloneFixtureNames(names []GoldenCoinFixtureName) []GoldenCoinFixtureName {
	if names == nil {
		return nil
	}
	cloned := make([]GoldenCoinFixtureName, len(names))
	copy(cloned, names)
	return cloned
}

func cloneTraits(traits []GoldenCoinTrait) []GoldenCoinTrait {
	if traits == nil {
		return nil
	}
	cloned := make([]GoldenCoinTrait, len(traits))
	copy(cloned, traits)
	return cloned
}

func cloneFloatPtr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}

func cloneUintPtr(v *uint) *uint {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}

func cloneTimePtr(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}
