package repository

import "github.com/briandenicola/ancient-coins-api/models"

// PageForLifestore returns up to limit of the user's coins with an id above
// afterID, in id order, with what the life store feed embeds. Private
// attribution cards are left out of the images, as everywhere else.
func (r *CoinRepository) PageForLifestore(userID, afterID uint, limit int) ([]models.Coin, error) {
	var coins []models.Coin
	err := r.db.Scopes(OwnedBy(userID)).
		Where("id > ?", afterID).
		Order("id ASC").
		Limit(limit).
		Preload("Images", CoinPhotos).
		Preload("Tags").
		Preload("References").
		Preload("Sets").
		Preload("StorageLocation").
		Preload("MintLocation").
		Find(&coins).Error
	return coins, err
}

// JournalsForCoins returns the user's journal entries for the given coins,
// oldest first.
func (r *CoinRepository) JournalsForCoins(userID uint, coinIDs []uint) ([]models.CoinJournal, error) {
	var entries []models.CoinJournal
	if len(coinIDs) == 0 {
		return entries, nil
	}
	err := r.db.Where("user_id = ? AND coin_id IN ?", userID, coinIDs).
		Order("created_at ASC, id ASC").
		Find(&entries).Error
	return entries, err
}

// ValueHistoryForCoins returns the user's recorded values for the given
// coins, oldest first.
func (r *CoinRepository) ValueHistoryForCoins(userID uint, coinIDs []uint) ([]models.CoinValueHistory, error) {
	var history []models.CoinValueHistory
	if len(coinIDs) == 0 {
		return history, nil
	}
	err := r.db.Where("user_id = ? AND coin_id IN ?", userID, coinIDs).
		Order("recorded_at ASC, id ASC").
		Find(&history).Error
	return history, err
}
