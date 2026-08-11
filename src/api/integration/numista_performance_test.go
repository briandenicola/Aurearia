package integration_test

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
)

type workloadClock struct {
	mu  sync.Mutex
	now time.Time
}

func newWorkloadClock() *workloadClock {
	return &workloadClock{now: time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)}
}
func (c *workloadClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}
func (c *workloadClock) Advance(duration time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(duration)
	c.mu.Unlock()
}

type workloadProvider struct {
	mu            sync.Mutex
	clock         *workloadClock
	searchLatency time.Duration
	detailLatency time.Duration
	searchCalls   int
	detailCalls   int
	candidates    []models.NumistaCandidate
	details       map[int]models.NumistaCandidate
}

func (p *workloadProvider) Search(context.Context, string, int) ([]models.NumistaCandidate, error) {
	p.mu.Lock()
	p.searchCalls++
	p.mu.Unlock()
	p.clock.Advance(p.searchLatency)
	return append([]models.NumistaCandidate(nil), p.candidates...), nil
}
func (p *workloadProvider) Detail(_ context.Context, id int) (models.NumistaCandidate, error) {
	p.mu.Lock()
	p.detailCalls++
	detail := p.details[id]
	p.mu.Unlock()
	p.clock.Advance(p.detailLatency)
	return detail, nil
}

func logicalPercentile(values []time.Duration, percentile float64) time.Duration {
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1) * percentile)
	return sorted[index]
}

func TestNumistaDeterministicWorkloadLatencyAndBroadCallReduction(t *testing.T) {
	clock := newWorkloadClock()
	provider := &workloadProvider{
		clock: clock, searchLatency: 4 * time.Second,
		candidates: []models.NumistaCandidate{{ID: 1, Title: "Trajan Denarius"}},
	}
	settings := &integrationNumistaSettings{
		key: "configured",
		config: services.NumistaSettings{
			SearchTTL: 24 * time.Hour, DetailTTL: 7 * 24 * time.Hour,
			SearchResultLimit: 20, EnrichmentLimit: 5, Valid: true,
		},
	}
	service := services.NewNumistaLookupService(
		provider, services.NewNumistaCache(clock, 500, 5000),
		services.NewNumistaV1Scorer(), services.NewNumistaTelemetry(500),
		settings, clock,
	)

	uncachedDurations := make([]time.Duration, 20)
	for index := range uncachedDurations {
		start := clock.Now()
		outcome, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
			Query:    fmt.Sprintf("unique Trajan workload %d", index),
			Path:     models.NumistaLookupPathDirect,
			Evidence: models.NumistaEvidence{Title: "Trajan Denarius"},
		})
		if err != nil || outcome.Status != models.NumistaStatusSuccess || outcome.Cache.Hit {
			t.Fatalf("uncached lookup %d outcome=%+v err=%v", index, outcome, err)
		}
		uncachedDurations[index] = clock.Now().Sub(start)
	}
	if p95 := logicalPercentile(uncachedDurations, 0.95); p95 > 5*time.Second {
		t.Fatalf("deterministic uncached p95=%v, budget=5s", p95)
	}

	const repeated = 25
	freshDurations := make([]time.Duration, 0, repeated-1)
	beforeCalls := provider.searchCalls
	for index := 0; index < repeated; index++ {
		start := clock.Now()
		outcome, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
			Query: "shared cached Trajan workload", Path: models.NumistaLookupPathPhoto,
			Evidence: models.NumistaEvidence{Title: "Trajan Denarius"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if index > 0 {
			if outcome.Cache == nil || !outcome.Cache.Hit {
				t.Fatalf("iteration %d was not a fresh cache hit: %+v", index, outcome.Cache)
			}
			freshDurations = append(freshDurations, clock.Now().Sub(start))
		}
	}
	broadCalls := provider.searchCalls - beforeCalls
	reduction := 1 - float64(broadCalls)/repeated
	if reduction < 0.80 {
		t.Fatalf("broad call reduction=%.1f%% calls=%d workload=%d", reduction*100, broadCalls, repeated)
	}
	if p95 := logicalPercentile(freshDurations, 0.95); p95 > time.Second {
		t.Fatalf("deterministic fresh-cache p95=%v, budget=1s", p95)
	}
}

func TestNumistaDeterministicScoringFixturesAndDefaultDetailCeiling(t *testing.T) {
	catalog := knownCoinCatalog()
	fixtures := knownCoinRankingFixtures()
	if len(fixtures) < 20 {
		t.Fatalf("SC-002 benchmark has %d fixtures, want at least 20", len(fixtures))
	}
	topThree := 0
	failures := make([]string, 0)
	for fixtureIndex, fixture := range fixtures {
		if fixture.evidence.ExactNumistaID != nil {
			t.Fatalf("%s leaks the expected answer through ExactNumistaID", fixture.name)
		}
		if len(fixture.candidateIDs) < 6 || !slices.Contains(fixture.candidateIDs, fixture.correctID) {
			t.Fatalf("%s has invalid candidate pool: correct=%d pool=%v",
				fixture.name, fixture.correctID, fixture.candidateIDs)
		}
		permutations := deterministicCandidatePermutations(fixture.candidateIDs, fixtureIndex)
		fixturePassed := true
		for permutationIndex, candidateIDs := range permutations {
			clock := newWorkloadClock()
			details := make(map[int]models.NumistaCandidate, len(candidateIDs))
			broad := make([]models.NumistaCandidate, len(candidateIDs))
			for index, id := range candidateIDs {
				detail, ok := catalog[id]
				if !ok {
					t.Fatalf("%s references unknown catalog candidate %d", fixture.name, id)
				}
				details[id] = detail
				broad[index] = broadKnownCoinCandidate(detail, index)
			}
			provider := &workloadProvider{
				clock: clock, detailLatency: 100 * time.Millisecond,
				candidates: broad, details: details,
			}
			settings := &integrationNumistaSettings{
				key: "configured",
				config: services.NumistaSettings{
					SearchTTL: time.Hour, DetailTTL: time.Hour, SearchResultLimit: 20,
					EnrichmentLimit: 0, Valid: true,
				},
			}
			service := services.NewNumistaLookupService(
				provider, services.NewNumistaCache(clock, 50, 50),
				services.NewNumistaV1Scorer(), services.NewNumistaTelemetry(50), settings, clock,
			)
			request := models.NumistaEnrichmentRequest{
				NumistaLookupRequest: models.NumistaLookupRequest{
					Query: fixture.query, Path: models.NumistaLookupPathDirect, Evidence: fixture.evidence,
				},
				Candidates: broad,
			}
			outcome, err := service.Enrich(context.Background(), request)
			if err != nil {
				t.Fatalf("%s permutation %d: %v", fixture.name, permutationIndex, err)
			}
			if provider.detailCalls > 5 {
				t.Fatalf("%s permutation %d exceeded default five-detail ceiling: calls=%d",
					fixture.name, permutationIndex, provider.detailCalls)
			}
			rank := candidateRank(outcome.Candidates, fixture.correctID)
			if rank < 1 || rank > 3 {
				fixturePassed = false
				failures = append(failures, rankingDiagnostic(
					fixture, permutationIndex, candidateIDs, rank, outcome.Candidates,
				))
			} else {
				assertCorrectCandidateUsesEvidence(t, fixture, permutationIndex, outcome.Candidates[rank-1])
				if len(outcome.Candidates) >= 4 &&
					outcome.Candidates[rank-1].Assessment.Score <= outcome.Candidates[3].Assessment.Score {
					t.Fatalf(
						"%s permutation %d is not mutation-sensitive: correct score=%d fourth score=%d",
						fixture.name, permutationIndex,
						outcome.Candidates[rank-1].Assessment.Score,
						outcome.Candidates[3].Assessment.Score,
					)
				}
			}
		}
		if fixturePassed {
			topThree++
		}
	}
	rate := float64(topThree) / float64(len(fixtures))
	if rate < 0.85 {
		t.Fatalf(
			"SC-002 correct-candidate top-three rate=%.1f%% (%d/%d), want >=85%%\n%s",
			rate*100, topThree, len(fixtures), strings.Join(failures, "\n"),
		)
	}
	if len(failures) > 0 {
		t.Logf(
			"SC-002 benchmark passed at %.1f%% (%d/%d); below-top-three diagnostics:\n%s",
			rate*100, topThree, len(fixtures), strings.Join(failures, "\n"),
		)
	} else {
		t.Logf(
			"SC-002 benchmark passed at %.1f%% (%d/%d): six candidates and three deterministic permutations per fixture",
			rate*100, topThree, len(fixtures),
		)
	}
}

type knownCoinRankingFixture struct {
	name         string
	query        string
	evidence     models.NumistaEvidence
	correctID    int
	candidateIDs []int
}

func knownCoinCandidate(
	id int,
	title, issuer, denomination, mint string,
	minYear, maxYear int,
	material, obverse, reverse string,
) models.NumistaCandidate {
	canonical, _ := models.CanonicalNumistaURL(id)
	return models.NumistaCandidate{
		ID: id, CanonicalURL: canonical, Title: title, Issuer: issuer,
		Denomination: denomination, Mint: mint, MinYear: &minYear, MaxYear: &maxYear,
		Material: material, ObverseInscription: obverse, ReverseInscription: reverse,
		EnrichmentState: models.NumistaEnrichmentEnriched,
	}
}

func knownCoinCatalog() map[int]models.NumistaCandidate {
	coins := []models.NumistaCandidate{
		knownCoinCandidate(1001, "Trajan Denarius - Victory", "Trajan Roman Empire", "Denarius", "Rome", 103, 111, "Silver", "IMP TRAIANO AVG GER DAC PM TRP", "COS V PP SPQR OPTIMO PRINC"),
		knownCoinCandidate(1002, "Hadrian Denarius - Pax", "Hadrian Roman Empire", "Denarius", "Rome", 117, 138, "Silver", "HADRIANVS AVG COS III PP", "PAX"),
		knownCoinCandidate(1003, "Vespasian Denarius - Judaea", "Vespasian Roman Empire", "Denarius", "Rome", 69, 79, "Silver", "IMP CAES VESP AVG", "IVDAEA"),
		knownCoinCandidate(1004, "Domitian Denarius - Minerva", "Domitian Roman Empire", "Denarius", "Rome", 81, 96, "Silver", "IMP CAES DOMIT AVG", "MINERVA"),
		knownCoinCandidate(1005, "Marcus Aurelius Denarius - Providentia", "Marcus Aurelius Roman Empire", "Denarius", "Rome", 161, 180, "Silver", "M ANTONINVS AVG ARM PARTH MAX", "TRP XXI IMP IIII COS III"),
		knownCoinCandidate(1006, "Septimius Severus Denarius - Victory", "Septimius Severus Roman Empire", "Denarius", "Rome", 193, 211, "Silver", "SEVERVS PIVS AVG", "VICT PART MAX"),
		knownCoinCandidate(1007, "Caracalla Antoninianus - Serapis", "Caracalla Roman Empire", "Antoninianus", "Rome", 215, 217, "Silver", "ANTONINVS PIVS AVG GERM", "PM TRP XVIII COS IIII PP"),
		knownCoinCandidate(1008, "Gordian III Antoninianus - Providentia", "Gordian III Roman Empire", "Antoninianus", "Rome", 238, 244, "Silver", "IMP GORDIANVS PIVS FEL AVG", "PROVIDENTIA AVG"),
		knownCoinCandidate(1009, "Aurelian Antoninianus - Sol", "Aurelian Roman Empire", "Antoninianus", "Siscia", 270, 275, "Billon", "IMP AVRELIANVS AVG", "ORIENS AVG"),
		knownCoinCandidate(1010, "Diocletian Follis - Genius", "Diocletian Roman Empire", "Follis", "Alexandria", 296, 305, "Bronze", "IMP C DIOCLETIANVS PF AVG", "GENIO POPVLI ROMANI"),
		knownCoinCandidate(1011, "Constantine I Follis - Campgate", "Constantine I Roman Empire", "Follis", "Trier", 324, 330, "Bronze", "CONSTANTINVS AVG", "PROVIDENTIAE AVGG"),
		knownCoinCandidate(1012, "Licinius I Follis - Jupiter", "Licinius I Roman Empire", "Follis", "Nicomedia", 308, 324, "Bronze", "IMP LIC LICINIVS PF AVG", "IOVI CONSERVATORI"),
		knownCoinCandidate(1013, "Nero Sestertius - Roma", "Nero Roman Empire", "Sestertius", "Lugdunum", 64, 68, "Orichalcum", "NERO CLAVD CAESAR AVG GER PM TRP IMP PP", "ROMA SC"),
		knownCoinCandidate(1014, "Antoninus Pius Sestertius - Annona", "Antoninus Pius Roman Empire", "Sestertius", "Rome", 138, 161, "Orichalcum", "ANTONINVS AVG PIVS PP TRP", "ANNONA AVG"),
		knownCoinCandidate(1015, "Augustus Denarius - Gaius and Lucius", "Augustus Roman Empire", "Denarius", "Lugdunum", -2, 4, "Silver", "CAESAR AVGVSTVS DIVI F PATER PATRIAE", "AVGVSTI F COS DESIG PRINC IVVENT"),
		knownCoinCandidate(1016, "Athens Owl Tetradrachm", "Athens Attica", "Tetradrachm", "Athens", -454, -404, "Silver", "ATHENA", "AOE"),
		knownCoinCandidate(1017, "Alexander III Tetradrachm - Zeus", "Alexander III Macedon", "Tetradrachm", "Amphipolis", -325, -315, "Silver", "HERAKLES", "ALEXANDROU ZEUS"),
		knownCoinCandidate(1018, "Antiochos VII Tetradrachm - Athena", "Antiochos VII Seleucid Kingdom", "Tetradrachm", "Antioch", -138, -129, "Silver", "ANTIOCHOU", "ATHENA NIKEPHOROS"),
		knownCoinCandidate(1019, "Ptolemy II Bronze Diobol - Eagle", "Ptolemy II Egypt", "Diobol", "Alexandria", -285, -246, "Bronze", "ZEUS AMMON", "PTOLEMAIOU BASILEOS"),
		knownCoinCandidate(1020, "Rhodes Drachm - Rose", "Rhodes Caria", "Drachm", "Rhodes", -205, -190, "Silver", "HELIOS", "RODON"),
		knownCoinCandidate(1021, "Justinian I Follis Year 12", "Justinian I Byzantine Empire", "Follis", "Constantinople", 538, 539, "Bronze", "DN IVSTINIANVS PP AVG", "M ANNO XII"),
		knownCoinCandidate(1022, "Heraclius Follis - Three Standing Figures", "Heraclius Byzantine Empire", "Follis", "Constantinople", 610, 641, "Bronze", "HERACLIUS HERACLIUS CONSTANTINE", "M"),
		knownCoinCandidate(1023, "Basil II Anonymous Follis Class A2", "Basil II Byzantine Empire", "Anonymous Follis", "Constantinople", 976, 1025, "Bronze", "EMMANOVHL", "IHSUS XRISTUS BASILEU BASILE"),
		knownCoinCandidate(1024, "Bohemond III Denier - Antioch", "Bohemond III Principality of Antioch", "Denier", "Antioch", 1163, 1201, "Billon", "BOAMVNDVS", "ANTIOCHIA"),
	}
	result := make(map[int]models.NumistaCandidate, len(coins))
	for _, coin := range coins {
		result[coin.ID] = coin
	}
	return result
}

func knownCoinRankingFixtures() []knownCoinRankingFixture {
	return []knownCoinRankingFixture{
		{"Trajan victory denarius", "Trajan silver Victory coin Rome", models.NumistaEvidence{Title: "Trajan denarius Victory", Issuer: "Trajan", Denomination: "denarius", Mint: "Rome", DateText: "103-111 CE", Material: "silver", ObverseInscription: "IMP TRAIANO", ReverseInscription: "OPTIMO PRINC"}, 1001, []int{1002, 1001, 1003, 1005, 1006, 1015}},
		{"Hadrian Pax denarius", "Hadrian Pax silver denarius", models.NumistaEvidence{Title: "Hadrian denarius Pax", Issuer: "Hadrian", Denomination: "denarius", Mint: "Rome", DateText: "117-138 CE", Material: "silver", VisibleText: "HADRIANVS PAX"}, 1002, []int{1001, 1005, 1002, 1003, 1004, 1015}},
		{"Vespasian Judaea denarius", "Vespasian Judaea captured denarius", models.NumistaEvidence{Title: "Vespasian denarius Judaea", Issuer: "Vespasian", Denomination: "denarius", DateText: "69-79 CE", Material: "silver", ReverseInscription: "IVDAEA"}, 1003, []int{1004, 1006, 1001, 1003, 1002, 1015}},
		{"Domitian Minerva denarius", "Domitian Minerva denarius Rome", models.NumistaEvidence{Title: "Domitian denarius Minerva", Issuer: "Domitian", Denomination: "denarius", Mint: "Rome", DateText: "81-96 CE", Material: "silver", VisibleText: "DOMIT MINERVA"}, 1004, []int{1003, 1004, 1005, 1001, 1002, 1006}},
		{"Marcus Aurelius denarius", "Marcus Aurelius Providentia denarius", models.NumistaEvidence{Title: "Marcus Aurelius denarius Providentia", Issuer: "Marcus Aurelius", Denomination: "denarius", DateText: "161-180 CE", Material: "silver", ObverseInscription: "M ANTONINVS", ReverseInscription: "TRP XXI"}, 1005, []int{1006, 1002, 1005, 1001, 1004, 1008}},
		{"Septimius Severus victory denarius", "Severus victory Parthia silver", models.NumistaEvidence{Title: "Septimius Severus denarius Victory", Issuer: "Septimius Severus", Denomination: "denarius", Mint: "Rome", DateText: "193-211 CE", Material: "silver", ReverseInscription: "VICT PART MAX"}, 1006, []int{1005, 1007, 1008, 1006, 1001, 1002}},
		{"Caracalla Serapis antoninianus", "Caracalla radiate Serapis coin", models.NumistaEvidence{Title: "Caracalla antoninianus Serapis", Issuer: "Caracalla", Denomination: "antoninianus", DateText: "215-217 CE", Material: "silver", ObverseInscription: "ANTONINVS PIVS", ReverseInscription: "TRP XVIII"}, 1007, []int{1008, 1009, 1007, 1006, 1005, 1010}},
		{"Gordian III Providentia antoninianus", "Gordian III Providentia antoninianus Rome", models.NumistaEvidence{Title: "Gordian III antoninianus Providentia", Issuer: "Gordian III", Denomination: "antoninianus", Mint: "Rome", DateText: "238-244 CE", Material: "silver", ReverseInscription: "PROVIDENTIA AVG"}, 1008, []int{1007, 1009, 1008, 1006, 1010, 1011}},
		{"Aurelian Sol antoninianus", "Aurelian Sol Oriens Siscia", models.NumistaEvidence{Title: "Aurelian antoninianus Sol", Issuer: "Aurelian", Denomination: "antoninianus", Mint: "Siscia", DateText: "270-275 CE", Material: "billon", ReverseInscription: "ORIENS AVG"}, 1009, []int{1008, 1007, 1010, 1009, 1011, 1012}},
		{"Diocletian Genius follis", "Diocletian large bronze Genius Alexandria", models.NumistaEvidence{Title: "Diocletian follis Genius", Issuer: "Diocletian", Denomination: "follis", Mint: "Alexandria", DateText: "296-305 CE", Material: "bronze", ReverseInscription: "GENIO POPVLI ROMANI"}, 1010, []int{1012, 1011, 1010, 1009, 1021, 1022}},
		{"Constantine campgate follis", "Constantine camp gate Trier bronze", models.NumistaEvidence{Title: "Constantine I follis campgate", Issuer: "Constantine I", Denomination: "follis", Mint: "Trier", DateText: "324-330 CE", Material: "bronze", ReverseInscription: "PROVIDENTIAE AVGG"}, 1011, []int{1012, 1010, 1011, 1009, 1021, 1022}},
		{"Licinius Jupiter follis", "Licinius Jupiter conservator Nicomedia", models.NumistaEvidence{Title: "Licinius I follis Jupiter", Issuer: "Licinius I", Denomination: "follis", Mint: "Nicomedia", DateText: "308-324 CE", Material: "bronze", ReverseInscription: "IOVI CONSERVATORI"}, 1012, []int{1011, 1010, 1009, 1012, 1021, 1022}},
		{"Nero Roma sestertius", "Nero sestertius Roma Lugdunum", models.NumistaEvidence{Title: "Nero sestertius Roma", Issuer: "Nero", Denomination: "sestertius", Mint: "Lugdunum", DateText: "64-68 CE", Material: "orichalcum", ReverseInscription: "ROMA SC"}, 1013, []int{1014, 1013, 1003, 1004, 1015, 1002}},
		{"Antoninus Pius Annona sestertius", "Antoninus Pius Annona brass sestertius", models.NumistaEvidence{Title: "Antoninus Pius sestertius Annona", Issuer: "Antoninus Pius", Denomination: "sestertius", Mint: "Rome", DateText: "138-161 CE", Material: "orichalcum", ReverseInscription: "ANNONA AVG"}, 1014, []int{1013, 1005, 1014, 1002, 1006, 1011}},
		{"Augustus Gaius Lucius denarius", "Augustus Gaius Lucius Lugdunum denarius", models.NumistaEvidence{Title: "Augustus denarius Gaius Lucius", Issuer: "Augustus", Denomination: "denarius", Mint: "Lugdunum", DateText: "2 BCE-4 CE", Material: "silver", ObverseInscription: "CAESAR AVGVSTVS", ReverseInscription: "PRINC IVVENT"}, 1015, []int{1003, 1001, 1015, 1002, 1004, 1013}},
		{"Athens owl tetradrachm", "Attica Athens owl Athena tetradrachm", models.NumistaEvidence{Title: "Athens owl tetradrachm", Issuer: "Athens Attica", Denomination: "tetradrachm", Mint: "Athens", DateText: "454-404 BCE", Material: "silver", VisibleText: "AOE"}, 1016, []int{1017, 1018, 1020, 1016, 1019, 1015}},
		{"Alexander Zeus tetradrachm", "Alexander III Herakles Zeus tetradrachm", models.NumistaEvidence{Title: "Alexander III tetradrachm Zeus", Issuer: "Alexander III Macedon", Denomination: "tetradrachm", Mint: "Amphipolis", DateText: "325-315 BCE", Material: "silver", ReverseInscription: "ALEXANDROU ZEUS"}, 1017, []int{1016, 1018, 1017, 1020, 1019, 1015}},
		{"Antiochos VII Athena tetradrachm", "Seleucid Antiochos Athena Nikephoros", models.NumistaEvidence{Title: "Antiochos VII tetradrachm Athena", Issuer: "Antiochos VII Seleucid", Denomination: "tetradrachm", Mint: "Antioch", DateText: "138-129 BCE", Material: "silver", ReverseInscription: "ATHENA NIKEPHOROS"}, 1018, []int{1017, 1016, 1020, 1018, 1019, 1024}},
		{"Ptolemy II eagle diobol", "Ptolemy II bronze eagle Alexandria", models.NumistaEvidence{Title: "Ptolemy II diobol eagle", Issuer: "Ptolemy II Egypt", Denomination: "diobol", Mint: "Alexandria", DateText: "285-246 BCE", Material: "bronze", ReverseInscription: "PTOLEMAIOU BASILEOS"}, 1019, []int{1018, 1017, 1019, 1020, 1016, 1010}},
		{"Rhodes rose drachm", "Rhodes Helios rose silver drachm", models.NumistaEvidence{Title: "Rhodes drachm rose", Issuer: "Rhodes Caria", Denomination: "drachm", Mint: "Rhodes", DateText: "205-190 BCE", Material: "silver", ObverseInscription: "HELIOS", ReverseInscription: "RODON"}, 1020, []int{1016, 1017, 1018, 1020, 1019, 1015}},
		{"Justinian year 12 follis", "Justinian follis year twelve Constantinople", models.NumistaEvidence{Title: "Justinian I follis year 12", Issuer: "Justinian I Byzantine", Denomination: "follis", Mint: "Constantinople", DateText: "538-539 CE", Material: "bronze", ObverseInscription: "IVSTINIANVS", ReverseInscription: "ANNO XII"}, 1021, []int{1022, 1023, 1021, 1010, 1011, 1012}},
		{"Heraclius three figures follis", "Heraclius three standing figures bronze", models.NumistaEvidence{Title: "Heraclius follis three figures", Issuer: "Heraclius Byzantine", Denomination: "follis", Mint: "Constantinople", DateText: "610-641 CE", Material: "bronze", ObverseInscription: "HERACLIUS CONSTANTINE", ReverseInscription: "M"}, 1022, []int{1021, 1023, 1022, 1010, 1011, 1012}},
		{"Basil II anonymous follis", "anonymous follis Christ Basil II class A2", models.NumistaEvidence{Title: "Basil II anonymous follis class A2", Issuer: "Basil II Byzantine", Denomination: "anonymous follis", Mint: "Constantinople", DateText: "976-1025 CE", Material: "bronze", ObverseInscription: "EMMANOVHL", ReverseInscription: "IHSUS XRISTUS"}, 1023, []int{1022, 1021, 1023, 1024, 1011, 1010}},
		{"Bohemond Antioch denier", "Bohemond III Antioch cross denier", models.NumistaEvidence{Title: "Bohemond III denier Antioch", Issuer: "Bohemond III", Denomination: "denier", Mint: "Antioch", DateText: "1163-1201 CE", Material: "billon", ObverseInscription: "BOAMVNDVS", ReverseInscription: "ANTIOCHIA"}, 1024, []int{1023, 1018, 1024, 1021, 1020, 1009}},
	}
}

func broadKnownCoinCandidate(detail models.NumistaCandidate, providerPosition int) models.NumistaCandidate {
	return models.NumistaCandidate{
		ID: detail.ID, CanonicalURL: detail.CanonicalURL, Title: detail.Title,
		Issuer: detail.Issuer, MinYear: detail.MinYear, MaxYear: detail.MaxYear,
		ProviderPosition: providerPosition, EnrichmentState: models.NumistaEnrichmentNotRequested,
		Assessment: models.NumistaRelevanceAssessment{
			ScoringVersion: models.NumistaScoringVersion, Score: 50, Band: "weak",
			Reasons: []models.NumistaRelevanceReason{},
		},
	}
}

func deterministicCandidatePermutations(ids []int, fixtureIndex int) [][]int {
	original := slices.Clone(ids)
	reversed := slices.Clone(ids)
	slices.Reverse(reversed)
	rotated := slices.Clone(ids)
	offset := fixtureIndex%len(rotated) + 1
	rotated = append(rotated[offset:], rotated[:offset]...)
	return [][]int{original, reversed, rotated}
}

func candidateRank(candidates []models.NumistaCandidate, expectedID int) int {
	for index, candidate := range candidates {
		if candidate.ID == expectedID {
			return index + 1
		}
	}
	return 0
}

func assertCorrectCandidateUsesEvidence(
	t *testing.T,
	fixture knownCoinRankingFixture,
	permutationIndex int,
	candidate models.NumistaCandidate,
) {
	t.Helper()
	expectedFields := map[string]bool{"title": true}
	evidence := fixture.evidence
	for field, value := range map[string]string{
		"issuer": evidence.Issuer, "denomination": evidence.Denomination,
		"mint": evidence.Mint, "date": evidence.DateText, "material": evidence.Material,
		"inscription": strings.Join([]string{
			evidence.ObverseInscription, evidence.ReverseInscription, evidence.VisibleText,
		}, " "),
	} {
		if strings.TrimSpace(value) != "" {
			expectedFields[field] = true
		}
	}
	matched := make(map[string]bool, len(candidate.Assessment.Reasons))
	for _, reason := range candidate.Assessment.Reasons {
		if reason.Kind == models.NumistaReasonMatch {
			matched[reason.Field] = true
		}
	}
	for field := range expectedFields {
		if !matched[field] {
			t.Fatalf(
				"%s permutation %d correct candidate %d did not use %s evidence: %+v",
				fixture.name, permutationIndex, candidate.ID, field, candidate.Assessment.Reasons,
			)
		}
	}
}

func rankingDiagnostic(
	fixture knownCoinRankingFixture,
	permutationIndex int,
	inputIDs []int,
	rank int,
	candidates []models.NumistaCandidate,
) string {
	rows := make([]string, 0, len(candidates))
	for index, candidate := range candidates {
		reasonCodes := make([]string, len(candidate.Assessment.Reasons))
		for reasonIndex, reason := range candidate.Assessment.Reasons {
			reasonCodes[reasonIndex] = reason.Code
		}
		rows = append(rows, fmt.Sprintf(
			"%d:%d score=%d state=%s reasons=%s",
			index+1, candidate.ID, candidate.Assessment.Score,
			candidate.EnrichmentState, strings.Join(reasonCodes, ","),
		))
	}
	return fmt.Sprintf(
		"%s permutation=%d correct=%d rank=%d input=%v ranking=[%s]",
		fixture.name, permutationIndex, fixture.correctID, rank, inputIDs, strings.Join(rows, "; "),
	)
}
