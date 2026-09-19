package repository

import "gorm.io/gorm"

// RunDeepProposalTransaction supplies transaction-scoped repositories for the
// existing Deep proposal write bridge. The service remains HTTP/database
// agnostic while every selected coin/reference/job mutation shares one commit.
func (r *CoinRepository) RunDeepProposalTransaction(
	deep *DeepIdentificationRepository,
	references *CoinReferenceRepository,
	registry *CatalogRegistryRepository,
	fn func(
		coin *CoinRepository,
		deep *DeepIdentificationRepository,
		references *CoinReferenceRepository,
		registry *CatalogRegistryRepository,
	) error,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(
			&CoinRepository{db: tx},
			&DeepIdentificationRepository{db: tx},
			&CoinReferenceRepository{db: tx},
			&CatalogRegistryRepository{db: tx},
		)
	})
}
