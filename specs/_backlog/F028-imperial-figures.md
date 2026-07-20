# F028 — RomanImperialFigure dataset: curated reference list

Companion content-curation deliverable for
`specs/_backlog/F028-roman-emperor-collection-tracker.md` (first pass,
reviewed 2026-07-20 — still expected to evolve; see Open follow-ups at the
bottom). This is the source list task #30 (Go model + migration + seeding)
will convert into actual seed data. Columns match the proposed
`models.RomanImperialFigure` fields: **Role** (emperor | empress | caesar |
usurper | other), **Region** (west | east), **Dynasty**, **Reign**, a
best-guess **Rarity** tier (common | scarce | rare | very_rare — flagged
as draft, see Decisions below), and **Notes**.

## Decisions made in this pass (please confirm/override)

1. **Zeno is included**, full stop — you asked for this explicitly. His first
   reign starts in 474 (before the 476 cutoff), so the cutoff rule is now:
   *"an emperor is in scope if their reign **began** on or before 476 AD, even
   if it continued after."* Zeno's entry runs 474–491 (through his restored
   reign) rather than being truncated at 476. This also pulls in **Basiliscus**
   (the usurper who deposed Zeno 475–476, during Zeno's own in-scope reign).
   It does **not** pull in Anastasius I (r. 491–518) — his reign began after
   476, so he stays out.
2. **Julius Caesar has a place**, tagged `role: other` (not `emperor` and not
   `caesar` — that role tag is reserved for the later imperial junior-co-ruler
   rank, e.g. Crispus, and re-using it for Julius Caesar would be confusing).
   Dynasty tagged `Late Republic (precursor)`, "reign" given as his dictatorship
   window (49–44 BC) rather than a true reign. He never counts toward the
   emperor-completion stat, but a user can log a Julius Caesar coin against
   him instead of leaving it as free text or picking nothing.
3. **Region convention**, derived directly from the backlog card's own
   dynasty-scope text: every figure from Augustus through Theodosius I
   (i.e., the whole unified empire, 27 BC – 395 AD) is tagged `region: west`,
   even ones who mostly ruled/resided in the eastern provinces (e.g. Valens).
   `region: east` only starts being used for the definitively separate Eastern
   line from 395 onward (Arcadius and after). This matches how the card's own
   "Western Roman" bucket already runs Julio-Claudian all the way through
   "Theodosian (West)" as one continuous list.
4. **Co-emperors count as separate entries** (Lucius Verus, Geta, Caracalla,
   etc. each get their own row) — leaning into this now since they minted
   coinage under their own name/portrait, so from a *collect the coin* point
   of view they're the right unit. This is still one of the card's open
   questions, not fully closed — flagging again here for your sign-off.
5. **Caesar-rank figures who never acceded to Augustus** get `role: caesar`
   and their own entry (Crispus, Licinius II, Tetricus II, Delmatius, Gallus).
   Figures who *started* as Caesar but later became Augustus (Constantine II,
   Constantius II, Constans, Julian) get **one** entry under `role: emperor`
   with their Caesar period mentioned in Notes, not a duplicate row.
6. **Usurper coverage is deliberately not exhaustive.** I included the
   usurpers with real, still-collectible coinage and historical significance
   (Gallic Empire, Palmyrene, Carausius/Allectus, the major 4th/5th-century
   Western pretenders). I did *not* include extremely obscure/disputed
   attributions (e.g. "Domitianus II", the disputed "Sponsian") — these would
   need actual numismatic sourcing before going in a seed table.
7. **Empress coverage is a starter set**, not guaranteed exhaustive — the
   well-known, commonly-collected wives/mothers/regents of emperors already
   in the list. Since empresses never count toward completion, getting this
   100% complete is lower-priority than getting the emperor list right; flag
   anyone you want added or cut.

Rough totals in this draft: **~96 emperor entries** (sole + co-emperors),
**~24 usurpers**, **~10 caesar-only entries**, **~34 empresses**, **1**
precursor (Julius Caesar) = **~165 rows** total, of which **~96 count toward
the emperor-completion stat**.

---

## Precursor (not counted toward completion)

| Name | Role | Region | Dynasty | Reign | Rarity | Notes |
|---|---|---|---|---|---|---|
| Julius Caesar | other | west | Late Republic (precursor) | 49–44 BC (dictator) | rare | Never "Emperor" — Augustus is the traditional first emperor. Included per explicit request so a coin can be logged against him without misusing the emperor list. |

## Julio-Claudian (27 BC – 68 AD)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Augustus | emperor | west | 27 BC – 14 AD | common | |
| Tiberius | emperor | west | 14–37 | common | |
| Caligula (Gaius) | emperor | west | 37–41 | scarce | |
| Claudius | emperor | west | 41–54 | common | |
| Nero | emperor | west | 54–68 | common | |

## Year of the Four Emperors (68–69 AD)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Galba | emperor | west | 68–69 | scarce | |
| Otho | emperor | west | 69 | rare | Mostly Rome-mint denarii; no provincial bronze. |
| Vitellius | emperor | west | 69 | scarce | |

## Flavian (69–96 AD)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Vespasian | emperor | west | 69–79 | common | |
| Titus | emperor | west | 79–81 | scarce | |
| Domitian | emperor | west | 81–96 | common | |

## Nerva–Antonine (96–192 AD)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Nerva | emperor | west | 96–98 | scarce | |
| Trajan | emperor | west | 98–117 | common | |
| Hadrian | emperor | west | 117–138 | common | |
| Antoninus Pius | emperor | west | 138–161 | common | |
| Marcus Aurelius | emperor | west | 161–180 | common | |
| Lucius Verus | emperor | west | 161–169 | scarce | Co-emperor with Marcus Aurelius. |
| Commodus | emperor | west | 177–192 | common | Co-emperor from 177, sole from 180. |

## Year of the Five Emperors (193 AD) + Severan (193–235 AD)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Pertinax | emperor | west | 193 | rare | |
| Didius Julianus | emperor | west | 193 | rare | |
| Pescennius Niger | usurper | west | 193–194 | rare | Rival claimant in the East, defeated by Severus. |
| Clodius Albinus | usurper | west | 193–197 | scarce | Rival claimant, Caesar then self-styled Augustus. |
| Septimius Severus | emperor | west | 193–211 | common | |
| Caracalla | emperor | west | 198–217 | common | Co-emperor from 198, sole from 211. |
| Geta | emperor | west | 209–211 | scarce | Co-emperor, murdered by Caracalla. |
| Macrinus | emperor | west | 217–218 | scarce | |
| Diadumenian | caesar | west | 218 | rare | Briefly hailed Augustus by troops just before his death; catalogued mainly as Caesar. |
| Elagabalus | emperor | west | 218–222 | common | |
| Severus Alexander | emperor | west | 222–235 | common | |

## Crisis of the Third Century (235–284 AD)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Maximinus Thrax | emperor | west | 235–238 | scarce | |
| Gordian I | emperor | west | 238 | very_rare | ~3-week reign. |
| Gordian II | emperor | west | 238 | very_rare | ~3-week reign, co-ruled with father. |
| Pupienus | emperor | west | 238 | scarce | |
| Balbinus | emperor | west | 238 | scarce | |
| Gordian III | emperor | west | 238–244 | common | |
| Philip I (the Arab) | emperor | west | 244–249 | common | |
| Philip II | emperor | west | 247–249 | scarce | Co-emperor (Caesar 244–247, Augustus 247–249). |
| Trajan Decius | emperor | west | 249–251 | common | |
| Herennius Etruscus | emperor | west | 251 | scarce | Co-emperor, killed alongside his father. |
| Hostilian | emperor | west | 251 | rare | Briefly sole/co-emperor; died of plague within months. |
| Trebonianus Gallus | emperor | west | 251–253 | scarce | |
| Volusianus | emperor | west | 251–253 | scarce | Co-emperor (son of Trebonianus Gallus). |
| Aemilian | emperor | west | 253 | rare | ~3-month reign. |
| Valerian | emperor | west | 253–260 | common | |
| Gallienus | emperor | west | 253–268 | common | Co-emperor 253–260, sole 260–268. |
| Saloninus | caesar | west | 260 | rare | Briefly hailed Augustus just before his death; catalogued mainly as Caesar. |
| Claudius II Gothicus | emperor | west | 268–270 | common | |
| Quintillus | emperor | west | 270 | scarce | |
| Aurelian | emperor | west | 270–275 | common | |
| Tacitus | emperor | west | 275–276 | scarce | |
| Florian | emperor | west | 276 | scarce | |
| Probus | emperor | west | 276–282 | common | |
| Carus | emperor | west | 282–283 | scarce | |
| Carinus | emperor | west | 283–285 | scarce | |
| Numerian | emperor | west | 283–284 | scarce | |

### Breakaway regimes (usurpers, 260s–290s)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Postumus | usurper | west | 260–269 | scarce | Gallic Empire. |
| Laelian | usurper | west | 269 | very_rare | Gallic Empire, brief. |
| Marius | usurper | west | 269 | very_rare | Gallic Empire, reportedly days-long reign. |
| Victorinus | usurper | west | 269–271 | scarce | Gallic Empire. |
| Tetricus I | usurper | west | 271–274 | scarce | Gallic Empire, last of the line. |
| Tetricus II | caesar | west | 271–274 | scarce | Gallic Empire, never sole Augustus. |
| Regalianus | usurper | west | 260 | very_rare | Brief Illyricum usurper under Gallienus. |
| Vabalathus | usurper | east | 267/270–272 | rare | Palmyrene Empire; briefly styled Augustus. |
| Zenobia | empress | east | 267–272 | rare | Palmyrene regent, styled Augusta. |
| Carausius | usurper | west | 286–293 | scarce | Britannic Empire; comparatively plentiful/popular with collectors. |
| Allectus | usurper | west | 293–296 | scarce | Britannic Empire, succeeded Carausius. |

## Tetrarchic/Diocletianic (284–306 AD)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Diocletian | emperor | west | 284–305 | common | Senior Augustus; region tag follows this draft's pre-395 "west" convention even though he was based in the East. |
| Maximian | emperor | west | 286–305, 306–308 | common | |
| Constantius I "Chlorus" | emperor | west | 293–306 | scarce | Caesar 293–305, Augustus 305–306. |
| Galerius | emperor | west | 293–311 | scarce | Caesar 293–305, Augustus 305–311. |
| Severus II | emperor | west | 306–307 | rare | |
| Maxentius | usurper | west | 306–312 | scarce | Held Rome/Italy against Constantine; not part of the recognized Tetrarchy succession. |
| Maximinus Daza (Daia) | emperor | west | 305–313 | scarce | Caesar 305–310, Augustus 310–313. |
| Licinius | emperor | west | 308–324 | common | |
| Licinius II | caesar | west | 317–324 | rare | Son of Licinius, executed 325, never Augustus. |

## Constantinian (306–363 AD)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Constantine I "the Great" | emperor | west | 306–337 | common | Sole ruler from 324. |
| Crispus | caesar | west | 317–326 | scarce | Executed by his father; never Augustus. |
| Constantine II | emperor | west | 337–340 | scarce | Caesar 317–337, Augustus 337–340. |
| Constantius II | emperor | west | 337–361 | common | Caesar 324–337, Augustus 337–361. |
| Constans | emperor | west | 337–350 | common | Caesar 333–337, Augustus 337–350. |
| Delmatius | caesar | west | 335–337 | very_rare | Executed 337; never Augustus. |
| Hannibalianus | other | west | 335–337 | very_rare | Held the special title "King of Kings," not standard Caesar rank; executed 337. |
| Magnentius | usurper | west | 350–353 | scarce | |
| Vetranio | usurper | west | 350 | rare | Briefly proclaimed Augustus, abdicated same year. |
| Nepotianus | usurper | west | 350 | very_rare | Rome-based usurper, reigned ~4 weeks. |
| Constantius Gallus | caesar | west | 351–354 | rare | Executed; never Augustus. |
| Julian "the Apostate" | emperor | west | 360–363 | common | Caesar 355–360, Augustus 360–363. |
| Jovian | emperor | west | 363–364 | scarce | |

## Valentinianic (364–392 AD)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Valentinian I | emperor | west | 364–375 | common | |
| Valens | emperor | west | 364–378 | common | Ruled the eastern provinces, but tagged `west` per this draft's pre-395 convention (see Decision 3). |
| Procopius | usurper | west | 365–366 | rare | Usurped against Valens in Constantinople. |
| Gratian | emperor | west | 367–383 | scarce | Co-emperor from 367, senior from 375. |
| Valentinian II | emperor | west | 375–392 | scarce | |
| Magnus Maximus | usurper | west | 383–388 | scarce | Usurped against Gratian; briefly controlled Britain/Gaul/Spain. |
| Eugenius | usurper | west | 392–394 | rare | Usurped against Valentinian II/Theodosius I. |

## Theodosian — West (393–455 AD)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Theodosius I "the Great" | emperor | west | 379–395 | common | Last sole ruler of the whole (still-unified) empire; region tagged `west` per Decision 3, even though his reign began in the East. |
| Honorius | emperor | west | 393–423 | common | |
| Constantius III | emperor | west | 421 | very_rare | Co-emperor for ~7 months; husband of Galla Placidia. |
| Valentinian III | emperor | west | 425–455 | scarce | |

## Western collapse (455–476 AD)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Petronius Maximus | emperor | west | 455 | very_rare | ~3-month reign. |
| Avitus | emperor | west | 455–456 | rare | |
| Priscus Attalus | usurper | west | 409–410, 414–415 | very_rare | Puppet emperor twice under Visigothic backing. |
| Constantine III | usurper | west | 407–411 | rare | British usurper who crossed to Gaul/Spain. |
| Jovinus | usurper | west | 411–413 | very_rare | Gallic usurper. |
| Sebastianus | usurper | west | 412–413 | very_rare | Co-usurper with Jovinus (his brother). |
| Maximus of Hispania | usurper | west | 409–411, 419–422 | very_rare | Spanish usurper, two separate spells. |
| Majorian | emperor | west | 457–461 | scarce | |
| Libius Severus (Severus III) | emperor | west | 461–465 | rare | |
| Anthemius | emperor | west | 467–472 | scarce | |
| Olybrius | emperor | west | 472 | very_rare | ~7-month reign. |
| Glycerius | emperor | west | 473–474 | very_rare | |
| Julius Nepos | emperor | west | 474–475 | rare | Continued to claim the title from Dalmatia until 480; only 474–475 counted as ruling from Italy. |
| Romulus Augustulus | emperor | west | 475–476 | very_rare | Last Western emperor recognized in Italy; deposed by Odoacer. |

## Theodosian — East (395–457 AD)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Arcadius | emperor | east | 395–408 | common | Co-emperor from 383, sole Eastern ruler from the 395 split. |
| Theodosius II | emperor | east | 408–450 | common | |
| Marcian | emperor | east | 450–457 | scarce | Married Pulcheria; not blood-Theodosian but conventionally grouped here. |

## Leonid (457–491 AD, capped per Decision 1)

| Name | Role | Region | Reign | Rarity | Notes |
|---|---|---|---|---|---|
| Leo I | emperor | east | 457–474 | common | |
| Leo II | emperor | east | 473–474 | very_rare | Co-emperor with grandfather Leo I, then sole for months. |
| Zeno | emperor | east | 474–491 | scarce | **Included per explicit request** — reign began 474 (before cutoff), continued past 476 via restoration; see Decision 1. |
| Basiliscus | usurper | east | 475–476 | rare | Deposed Zeno during his first reign; falls entirely within Zeno's in-scope reign window. |

---

## Notable empresses (starter set, not counted toward completion)

| Name | Region | Associated with | Rarity | Notes |
|---|---|---|---|---|
| Livia | west | Augustus | common | |
| Agrippina the Younger | west | Claudius/Nero | scarce | |
| Poppaea Sabina | west | Nero | rare | Mostly provincial issues. |
| Domitia | west | Domitian | scarce | |
| Plotina | west | Trajan | scarce | |
| Sabina | west | Hadrian | scarce | |
| Faustina the Elder | west | Antoninus Pius | common | |
| Faustina the Younger | west | Marcus Aurelius | common | |
| Lucilla | west | Lucius Verus | scarce | |
| Crispina | west | Commodus | scarce | |
| Julia Domna | west | Septimius Severus | common | |
| Plautilla | west | Caracalla | scarce | |
| Julia Maesa | west | grandmother of Elagabalus/Sev. Alexander | scarce | |
| Julia Soaemias | west | Elagabalus | scarce | |
| Julia Mamaea | west | Severus Alexander | scarce | |
| Otacilia Severa | west | Philip I | scarce | |
| Herennia Etruscilla | west | Trajan Decius | scarce | |
| Salonina | west | Gallienus | scarce | |
| Severina | west | Aurelian | rare | Possibly ruled briefly as regent, 275. |
| Magnia Urbica | west | Carinus | rare | |
| Galeria Valeria | west | Galerius | rare | Daughter of Diocletian. |
| Helena | west | mother of Constantine I | scarce | |
| Fausta | west | Constantine I | rare | |
| Theodora | west | Constantius Chlorus | very_rare | Coinage existence uncertain — needs verification. |
| Aelia Flaccilla | west | Theodosius I | rare | |
| Eudoxia | east | Arcadius | scarce | |
| Galla Placidia | west | regent for Valentinian III | scarce | |
| Pulcheria | east | sister/co-ruler, later wife of Marcian | scarce | Augusta from 414, well before Marcian. |
| Eudocia | east | Theodosius II | scarce | |
| Licinia Eudoxia | west | Valentinian III | rare | |
| Verina | east | Leo I | rare | |
| Ariadne | east | Zeno | rare | |

---

## Open follow-ups from this pass

- Confirm Decision 4 (co-emperors as separate entries) — this is the biggest
  scope lever; folding co-emperors into their senior partner would cut the
  emperor count by roughly 15–20 entries.
- Rarity tiers above are my best-guess draft, not sourced against real
  auction-frequency data — matches the card's existing open question about
  who curates `rarityTier` and against what standard.
- A few empress coinage attributions (Theodora especially) need verification
  before being treated as confirmed rather than "probably existed."
- Let me know which usurpers you'd rather cut or add — I erred toward
  "significant and still collectible" rather than exhaustive.
