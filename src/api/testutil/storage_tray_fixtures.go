package testutil

import "github.com/briandenicola/ancient-coins-api/models"

func StandardLocation(userID uint, name string) models.StorageLocation {
	return models.StorageLocation{UserID: userID, Name: name, Type: models.StorageLocationTypeStandard}
}

func CoinTray(userID uint, name string, rows, columns int) models.StorageLocation {
	return models.StorageLocation{
		UserID: userID, Name: name, Type: models.StorageLocationTypeTray,
		Rows: &rows, Columns: &columns,
	}
}

func CoinInSlot(userID uint, name string, locationID uint, slot int) models.Coin {
	return models.Coin{UserID: userID, Name: name, StorageLocationID: &locationID, StorageSlot: &slot}
}

func MinimalTrayImage(coinID uint, filePath string) models.CoinImage {
	return models.CoinImage{CoinID: coinID, FilePath: filePath, ImageType: "obverse", IsPrimary: true}
}
