package services

func fixtureHealthWeights() HealthWeights {
	return DefaultHealthWeights()
}

func fixtureCollectionHealthSummary(score int, delta *int) CollectionHealthSummary {
	return CollectionHealthSummary{
		Score:             ClampHealthScore(score),
		Grade:             HealthGradeFromScore(score),
		EligibleCoinCount: 3,
		Weights:           fixtureHealthWeights(),
		Dimensions: HealthDimensionScores{
			Metadata:           ClampHealthScore(score),
			ImageCoverage:      ClampHealthScore(score),
			ValuationFreshness: ClampHealthScore(score),
			AICoverage:         ClampHealthScore(score),
		},
		Trend30D: HealthTrend30D{
			Status:    "available",
			Delta:     delta,
			Direction: "flat",
		},
	}
}

func fixtureCoinHealthItem(coinID uint, score int) CoinHealthItem {
	return CoinHealthItem{
		CoinID: coinID,
		Title:  "Fixture Coin",
		Score:  ClampHealthScore(score),
		Grade:  HealthGradeFromScore(score),
		Dimensions: HealthDimensionScores{
			Metadata:           ClampHealthScore(score),
			ImageCoverage:      ClampHealthScore(score),
			ValuationFreshness: ClampHealthScore(score),
			AICoverage:         ClampHealthScore(score),
		},
		MissingItems: []MissingChecklistItem{
			{
				Key:        "metadata.denomination",
				Dimension:  ChecklistDimensionMetadata,
				Label:      "Add denomination",
				Severity:   ChecklistSeverityMedium,
				ActionHint: HealthActionEditMetadata,
			},
		},
		QuickActions: []HealthActionHint{HealthActionEditMetadata},
	}
}
